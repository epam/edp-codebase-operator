package chain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutProject struct {
	k8sClient                   client.Client
	gerritClient                gerrit.Client
	gitProjectProvider          func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error)
	gitProviderFactory          gitproviderv2.GitProviderFactory
	createGitProviderWithConfig func(config gitproviderv2.Config) gitproviderv2.Git
}

var (
	skipPutProjectStatuses = []string{util.ProjectPushedStatus, util.ProjectGitLabCIPushedStatus, util.ProjectTemplatesPushedStatus}
	putProjectStrategies   = []codebaseApi.Strategy{codebaseApi.Clone, codebaseApi.Create}
)

func NewPutProject(
	c client.Client,
	gerritProvider gerrit.Client,
	gitProjectProvider func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error),
	gitProviderFactory gitproviderv2.GitProviderFactory,
	createGitProviderWithConfig func(config gitproviderv2.Config) gitproviderv2.Git,
) *PutProject {
	return &PutProject{
		k8sClient:                   c,
		gerritClient:                gerritProvider,
		gitProjectProvider:          gitProjectProvider,
		gitProviderFactory:          gitProviderFactory,
		createGitProviderWithConfig: createGitProviderWithConfig,
	}
}

// ServeRequest is a method to put project into git repository.
// TODO: Refactor this method to smaller methods. Currently it is too big and complex.
func (h *PutProject) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx).WithValues("projectID", codebase.Spec.GetProjectID())

	if h.skip(ctx, codebase) {
		return nil
	}

	log.Info("Start putting project", "spec", codebase.Spec)

	err := setIntermediateSuccessFields(ctx, h.k8sClient, codebase, codebaseApi.RepositoryProvisioning)
	if err != nil {
		return fmt.Errorf("failed to update Codebase %v status: %w", codebase.Name, err)
	}

	repoContext, err := GetGitRepositoryContext(ctx, h.k8sClient, codebase)
	if err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())

		return fmt.Errorf("failed to get git repository context: %w", err)
	}

	if err = util.CreateDirectory(repoContext.WorkDir); err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())

		return fmt.Errorf("failed to create dir %q: %w", repoContext.WorkDir, err)
	}

	err = h.initialProjectProvisioning(ctx, codebase, repoContext)
	if err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to perform initial provisioning of codebase %v: %w", codebase.Name, err)
	}

	if err = h.checkoutBranch(ctrl.LoggerInto(ctx, log), codebase, repoContext); err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())
		return err
	}

	err = h.createProject(ctrl.LoggerInto(ctx, log), codebase, repoContext)
	if err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to create project: %w", err)
	}

	if err = updateGitStatusWithPatch(ctx, h.k8sClient, codebase, codebaseApi.RepositoryProvisioning, util.ProjectPushedStatus); err != nil {
		return err
	}

	log.Info("Finish putting project")

	return nil
}

func (*PutProject) skip(ctx context.Context, codebase *codebaseApi.Codebase) bool {
	log := ctrl.LoggerFrom(ctx)

	if !slices.Contains(putProjectStrategies, codebase.Spec.Strategy) {
		log.Info("Skip putting project to repository for non-clone or non-create strategy")
		return true
	}

	if slices.Contains(skipPutProjectStatuses, codebase.Status.Git) {
		log.Info("Skipping putting project, it has been already pushed")
		return true
	}

	return false
}

func (h *PutProject) createProject(
	ctx context.Context,
	codebase *codebaseApi.Codebase,
	repoContext *GitRepositoryContext,
) error {
	g := h.gitProviderFactory(repoContext.GitServer, repoContext.GitServerSecret)

	if repoContext.GitServer.Spec.GitProvider == codebaseApi.GitProviderGerrit {
		err := h.createGerritProject(ctx, repoContext.GitServer, repoContext.PrivateSSHKey, codebase.Spec.GetProjectID())
		if err != nil {
			return fmt.Errorf("failed to create project in Gerrit for codebase %v: %w", codebase.Name, err)
		}
	} else {
		if err := h.createGitThirdPartyProject(ctx, repoContext.GitServer, repoContext.Token, codebase); err != nil {
			return err
		}
	}

	err := h.pushProject(ctx, g, codebase.Spec.GetProjectID(), repoContext)
	if err != nil {
		return err
	}

	err = h.setDefaultBranch(ctx, repoContext.GitServer, codebase, repoContext.Token, repoContext.PrivateSSHKey)
	if err != nil {
		return err
	}

	return nil
}

func (h *PutProject) replaceDefaultBranch(ctx context.Context, g gitproviderv2.Git, directory, defaultBranchName, newBranchName string) error {
	log := ctrl.LoggerFrom(ctx).
		WithValues("defaultBranch", defaultBranchName, "newBranch", newBranchName)

	log.Info("Replacing default branch with new one")
	log.Info("Removing default branch")

	if err := g.RemoveBranch(ctx, directory, defaultBranchName); err != nil {
		return fmt.Errorf("failed to remove master branch: %w", err)
	}

	log.Info("Creating new branch")

	if err := g.CreateChildBranch(ctx, directory, newBranchName, defaultBranchName); err != nil {
		return fmt.Errorf("failed to create child branch: %w", err)
	}

	log.Info("Branch has been successfully created")

	return nil
}

func (h *PutProject) pushProject(ctx context.Context, g gitproviderv2.Git, projectName string, repoContext *GitRepositoryContext) error {
	log := ctrl.LoggerFrom(ctx).WithValues("gitProvider", repoContext.GitServer.Spec.GitProvider)

	log.Info("Start pushing project")
	log.Info("Start adding remote link")

	if err := g.AddRemoteLink(
		ctx,
		repoContext.WorkDir,
		util.GetProjectGitUrl(repoContext.GitServer, repoContext.GitServerSecret, projectName),
	); err != nil {
		return fmt.Errorf("failed to add remote link: %w", err)
	}

	log.Info("Start pushing changes into git")

	if err := g.Push(ctx, repoContext.WorkDir, gitproviderv2.RefSpecPushAllBranches); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info("Start pushing tags into git")

	if err := g.Push(ctx, repoContext.WorkDir, gitproviderv2.RefSpecPushAllTags); err != nil {
		return fmt.Errorf("failed to push changes into git: %w", err)
	}

	log.Info("Project has been pushed successfully")

	return nil
}

func (h *PutProject) createGerritProject(ctx context.Context, gitServer *codebaseApi.GitServer, privateSSHKey, projectName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating project in Gerrit")

	projectExist, err := h.gerritClient.CheckProjectExist(gitServer.Spec.SshPort, privateSSHKey, gitServer.Spec.GitHost, gitServer.Spec.GitUser, projectName, log)
	if err != nil {
		return fmt.Errorf("failed to check if project exist in Gerrit: %w", err)
	}

	if projectExist {
		log.Info("Skip creating project in Gerrit, project already exist")
		return nil
	}

	err = h.gerritClient.CreateProject(gitServer.Spec.SshPort, privateSSHKey, gitServer.Spec.GitHost, gitServer.Spec.GitUser, projectName, log)
	if err != nil {
		return fmt.Errorf("failed to create gerrit project: %w", err)
	}

	log.Info("Project created in Gerrit")

	return nil
}

func (h *PutProject) checkoutBranch(ctx context.Context, codebase *codebaseApi.Codebase, repoContext *GitRepositoryContext) error {
	log := ctrl.LoggerFrom(ctx).WithValues(
		"defaultBranch",
		codebase.Spec.DefaultBranch,
		"branchToCopy",
		codebase.Spec.BranchToCopyInDefaultBranch,
	)

	g := h.gitProviderFactory(repoContext.GitServer, repoContext.GitServerSecret)

	repoUrl, err := util.GetRepoUrl(codebase)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	// TODO: branchToCopyInDefaultBranch is never used. Check if we can remove it.
	if codebase.Spec.BranchToCopyInDefaultBranch != "" && codebase.Spec.DefaultBranch != codebase.Spec.BranchToCopyInDefaultBranch {
		log.Info("Start checkout branch to copy")

		err = CheckoutBranch(ctx, repoUrl, repoContext.WorkDir, codebase.Spec.BranchToCopyInDefaultBranch, g, codebase, h.k8sClient, h.createGitProviderWithConfig)
		if err != nil {
			return fmt.Errorf("failed to checkout default branch %s: %w", codebase.Spec.DefaultBranch, err)
		}

		log.Info("Start replace default branch")

		err = h.replaceDefaultBranch(ctx, g, repoContext.WorkDir, codebase.Spec.DefaultBranch, codebase.Spec.BranchToCopyInDefaultBranch)
		if err != nil {
			return fmt.Errorf("failed to replace master: %w", err)
		}

		return nil
	}

	log.Info("Start checkout branch")

	err = CheckoutBranch(ctx, repoUrl, repoContext.WorkDir, codebase.Spec.DefaultBranch, g, codebase, h.k8sClient, h.createGitProviderWithConfig)
	if err != nil {
		return fmt.Errorf("failed to checkout default branch %s: %w", codebase.Spec.DefaultBranch, err)
	}

	log.Info("Checkout branch finished")

	return nil
}

func (h *PutProject) createGitThirdPartyProject(
	ctx context.Context,
	gitServer *codebaseApi.GitServer,
	gitProviderToken string,
	codebase *codebaseApi.Codebase,
) error {
	projectName := codebase.Spec.GetProjectID()
	log := ctrl.LoggerFrom(ctx).WithValues("gitProvider", gitServer.Spec.GitProvider)

	log.Info("Start creating project in git provider")

	gitProvider, err := h.gitProjectProvider(gitServer, gitProviderToken)
	if err != nil {
		return fmt.Errorf("failed to create git provider: %w", err)
	}

	projectExists, err := gitProvider.ProjectExists(
		ctx,
		gitprovider.GetGitProviderAPIURL(gitServer),
		gitProviderToken,
		projectName,
	)
	if err != nil {
		return fmt.Errorf("failed to check if project exists: %w", err)
	}

	if projectExists {
		log.Info("Skip creating project in git provider, project already exist")

		return nil
	}

	if err = gitProvider.CreateProject(
		ctx,
		gitprovider.GetGitProviderAPIURL(gitServer),
		gitProviderToken,
		projectName,
		gitprovider.RepositorySettings{
			IsPrivate: codebase.Spec.Private,
		},
	); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	log.Info("Project created in git provider")

	return nil
}

func (h *PutProject) setDefaultBranch(
	ctx context.Context,
	gitServer *codebaseApi.GitServer,
	codebase *codebaseApi.Codebase,
	gitProviderToken,
	privateSSHKey string,
) error {
	log := ctrl.LoggerFrom(ctx).
		WithValues("gitProvider", gitServer.Spec.GitProvider)

	log.Info("Start setting default branch", "defaultBranch", codebase.Spec.DefaultBranch)

	if gitServer.Spec.GitProvider == codebaseApi.GitProviderGerrit {
		log.Info("Set HEAD to default branch in Gerrit")

		err := h.gerritClient.SetHeadToBranch(
			gitServer.Spec.SshPort,
			privateSSHKey,
			gitServer.Spec.GitHost,
			gitServer.Spec.GitUser,
			codebase.Spec.GetProjectID(),
			codebase.Spec.DefaultBranch,
			log,
		)
		if err != nil {
			return fmt.Errorf(
				"set remote HEAD for codebase %s to default branch %s has been failed: %w",
				codebase.Spec.GetProjectID(),
				codebase.Spec.DefaultBranch,
				err,
			)
		}

		log.Info("Set HEAD to default branch in Gerrit has been finished")

		return nil
	}

	log.Info("Set default branch in git provider")

	gitProvider, err := h.gitProjectProvider(gitServer, gitProviderToken)
	if err != nil {
		return fmt.Errorf("failed to create git provider: %w", err)
	}

	if err = gitProvider.SetDefaultBranch(
		ctx,
		gitprovider.GetGitProviderAPIURL(gitServer),
		gitProviderToken,
		codebase.Spec.GetProjectID(),
		codebase.Spec.DefaultBranch,
	); err != nil {
		if errors.Is(err, gitprovider.ErrApiNotSupported) {
			// We can skip this error, because it is not supported by Git provider.
			// And this is not critical for the whole process.
			log.Info("Setting default branch is not supported by Git provider. Set it manually if needed")

			return nil
		}

		return fmt.Errorf("failed to set default branch: %w", err)
	}

	log.Info("Default branch has been set")

	return nil
}

func (h *PutProject) tryToCloneRepo(
	ctx context.Context,
	repoUrl string,
	repositoryUsername, repositoryPassword *string,
	repoContext *GitRepositoryContext,
) error {
	log := ctrl.LoggerFrom(ctx).WithValues("dest", repoContext.WorkDir, "repoUrl", repoUrl)

	log.Info("Start cloning repository")

	if util.DoesDirectoryExist(repoContext.WorkDir + "/.git") {
		log.Info("Repository already exists")

		return nil
	}

	config := gitproviderv2.Config{}

	if repositoryUsername != nil && repositoryPassword != nil {
		config = gitproviderv2.Config{}
		config.Username = *repositoryUsername
		config.Token = *repositoryPassword
	}

	g := h.createGitProviderWithConfig(config)
	if err := g.Clone(ctx, repoUrl, repoContext.WorkDir, 0); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info("Repository has been cloned")

	return nil
}

func (h *PutProject) squashCommits(ctx context.Context, workDir string, strategy codebaseApi.Strategy) error {
	log := ctrl.LoggerFrom(ctx).WithValues("dest", workDir, "strategy", strategy)

	if strategy != codebaseApi.Create {
		return nil
	}

	log.Info("Start squashing commits")

	err := os.RemoveAll(workDir + "/.git")
	if err != nil {
		return fmt.Errorf("failed to remove .git folder: %w", err)
	}

	g := h.createGitProviderWithConfig(gitproviderv2.Config{})

	if err := g.Init(ctx, workDir); err != nil {
		return fmt.Errorf("failed to create git repository: %w", err)
	}

	if err := g.Commit(ctx, workDir, "Initial commit"); err != nil {
		return fmt.Errorf("failed to commit all default content: %w", err)
	}

	log.Info("Commits have been squashed")

	return nil
}

func (h *PutProject) initialProjectProvisioning(ctx context.Context, codebase *codebaseApi.Codebase, repoContext *GitRepositoryContext) error {
	if codebase.Spec.EmptyProject {
		return h.emptyProjectProvisioning(ctx, repoContext)
	}

	return h.notEmptyProjectProvisioning(ctx, codebase, repoContext)
}

func (h *PutProject) emptyProjectProvisioning(ctx context.Context, repoContext *GitRepositoryContext) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Initialing empty git repository")

	g := h.createGitProviderWithConfig(gitproviderv2.Config{})

	if err := g.Init(ctx, repoContext.WorkDir); err != nil {
		return fmt.Errorf("failed to create empty git repository: %w", err)
	}

	log.Info("Making initial commit")

	if err := g.Commit(ctx, repoContext.WorkDir, "Initial commit", gitproviderv2.CommitAllowEmpty()); err != nil {
		return fmt.Errorf("failed to create Initial commit: %w", err)
	}

	return nil
}

func (h *PutProject) notEmptyProjectProvisioning(ctx context.Context, codebase *codebaseApi.Codebase, repoContext *GitRepositoryContext) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start initial provisioning for non-empty project")

	repoUrl, err := util.GetRepoUrl(codebase)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	repu, repp, err := GetRepositoryCredentialsIfExists(codebase, h.k8sClient)
	// we are ok if no credentials is found, assuming this is a public repo
	if err != nil && !k8sErrors.IsNotFound(err) {
		return fmt.Errorf("failed to get repository credentials: %w", err)
	}

	// Check permissions if credentials exist
	if repu != nil && repp != nil {
		tempConfig := gitproviderv2.Config{
			Username: *repu,
			Token:    *repp,
		}
		tempProvider := h.createGitProviderWithConfig(tempConfig)

		if err := tempProvider.CheckPermissions(ctx, repoUrl); err != nil {
			return fmt.Errorf("failed to get access to the repository %v for user %v: %w", repoUrl, *repu, err)
		}
	}

	if err = h.tryToCloneRepo(ctx, repoUrl, repu, repp, repoContext); err != nil {
		return fmt.Errorf("failed to clone template project: %w", err)
	}

	if err = h.squashCommits(ctx, repoContext.WorkDir, codebase.Spec.Strategy); err != nil {
		return fmt.Errorf("failed to squash commits in a template repo: %w", err)
	}

	return nil
}

func setFailedFields(c *codebaseApi.Codebase, a codebaseApi.ActionType, message string) {
	// Set WebHookRef from WebHookID for backward compatibility.
	webHookRef := c.Status.WebHookRef
	if webHookRef == "" && c.Status.WebHookID != 0 {
		webHookRef = strconv.Itoa(c.Status.WebHookID)
	}

	c.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusFailed,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Username:        "system",
		Action:          a,
		Result:          codebaseApi.Error,
		DetailedMessage: message,
		Value:           "failed",
		FailureCount:    c.Status.FailureCount,
		Git:             c.Status.Git,
		WebHookID:       c.Status.WebHookID,
		WebHookRef:      webHookRef,
		GitWebUrl:       c.Status.GitWebUrl,
	}
}
