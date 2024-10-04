package chain

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutProject struct {
	client             client.Client
	git                git.Git
	gerrit             gerrit.Client
	gitProjectProvider func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error)
}

var (
	skipPutProjectStatuses = []string{util.ProjectPushedStatus, util.ProjectTemplatesPushedStatus}
	putProjectStrategies   = []codebaseApi.Strategy{codebaseApi.Clone, codebaseApi.Create}
)

func NewPutProject(
	c client.Client,
	g git.Git,
	gerritProvider gerrit.Client,
	gitProjectProvider func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error),
) *PutProject {
	return &PutProject{client: c, git: g, gerrit: gerritProvider, gitProjectProvider: gitProjectProvider}
}

func (h *PutProject) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx).WithValues("projectID", codebase.Spec.GetProjectID())

	if h.skip(ctx, codebase) {
		return nil
	}

	log.Info("Start putting project", "spec", codebase.Spec)

	err := setIntermediateSuccessFields(ctx, h.client, codebase, codebaseApi.GerritRepositoryProvisioning)
	if err != nil {
		return fmt.Errorf("failed to update Codebase %v status: %w", codebase.Name, err)
	}

	wd := util.GetWorkDir(codebase.Name, codebase.Namespace)

	if err = util.CreateDirectory(wd); err != nil {
		setFailedFields(codebase, codebaseApi.GerritRepositoryProvisioning, err.Error())

		return fmt.Errorf("failed to create dir %q: %w", wd, err)
	}

	gitServer := &codebaseApi.GitServer{}
	if err = h.client.Get(
		ctx,
		client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace},
		gitServer,
	); err != nil {
		setFailedFields(codebase, codebaseApi.GerritRepositoryProvisioning, err.Error())

		return fmt.Errorf("failed to get GitServer %s: %w", codebase.Spec.GitServer, err)
	}

	err = h.initialProjectProvisioning(ctx, codebase, wd)
	if err != nil {
		setFailedFields(codebase, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to perform initial provisioning of codebase %v: %w", codebase.Name, err)
	}

	if err = h.checkoutBranch(ctrl.LoggerInto(ctx, log), codebase, wd); err != nil {
		setFailedFields(codebase, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return err
	}

	err = h.createProject(ctrl.LoggerInto(ctx, log), codebase, gitServer, wd)
	if err != nil {
		setFailedFields(codebase, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to create project: %w", err)
	}

	codebase.Status.Git = util.ProjectPushedStatus
	if err = h.client.Status().Update(ctx, codebase); err != nil {
		setFailedFields(codebase, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to set git status %s for codebase %s: %w", util.ProjectPushedStatus, codebase.Name, err)
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
	gitServer *codebaseApi.GitServer,
	workDir string,
) error {
	gitServerSecret := &corev1.Secret{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: gitServer.Spec.NameSshKeySecret, Namespace: codebase.Namespace}, gitServerSecret); err != nil {
		return fmt.Errorf("failed to get git server secret: %w", err)
	}

	privateSSHKey := string(gitServerSecret.Data[util.PrivateSShKeyName])
	gitProviderToken := string(gitServerSecret.Data[util.GitServerSecretTokenField])

	if gitServer.Spec.GitProvider == codebaseApi.GitProviderGerrit {
		err := h.createGerritProject(ctx, gitServer, privateSSHKey, codebase.Spec.GetProjectID())
		if err != nil {
			return fmt.Errorf("failed to create project in Gerrit for codebase %v: %w", codebase.Name, err)
		}
	} else {
		if err := h.createGitThirdPartyProject(ctx, gitServer, gitProviderToken, codebase.Spec.GetProjectID()); err != nil {
			return err
		}
	}

	err := h.pushProject(ctx, gitServer, privateSSHKey, codebase.Spec.GetProjectID(), workDir)
	if err != nil {
		return err
	}

	err = h.setDefaultBranch(ctx, gitServer, codebase, gitProviderToken, privateSSHKey)
	if err != nil {
		return err
	}

	return nil
}

func (h *PutProject) replaceDefaultBranch(ctx context.Context, directory, defaultBranchName, newBranchName string) error {
	log := ctrl.LoggerFrom(ctx).
		WithValues("defaultBranch", defaultBranchName, "newBranch", newBranchName)

	log.Info("Replacing default branch with new one")
	log.Info("Removing default branch")

	if err := h.git.RemoveBranch(directory, defaultBranchName); err != nil {
		return fmt.Errorf("failed to remove master branch: %w", err)
	}

	log.Info("Creating new branch")

	if err := h.git.CreateChildBranch(directory, newBranchName, defaultBranchName); err != nil {
		return fmt.Errorf("failed to create child branch: %w", err)
	}

	log.Info("Project has been successfully created")

	return nil
}

func (h *PutProject) pushProject(ctx context.Context, gitServer *codebaseApi.GitServer, privateSSHKey, projectName, directory string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("gitProvider", gitServer.Spec.GitProvider)

	log.Info("Start pushing project")
	log.Info("Start adding remote link to Gerrit")

	if err := h.git.AddRemoteLink(
		directory,
		util.GetSSHUrl(gitServer, projectName),
	); err != nil {
		return fmt.Errorf("failed to add remote link to Gerrit: %w", err)
	}

	log.Info("Start pushing changes into git")

	if err := h.git.PushChanges(privateSSHKey, gitServer.Spec.GitUser, directory, gitServer.Spec.SshPort, "--all"); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info("Start pushing tags into git")

	if err := h.git.PushChanges(privateSSHKey, gitServer.Spec.GitUser, directory, gitServer.Spec.SshPort, "--tags"); err != nil {
		return fmt.Errorf("failed to push changes into git: %w", err)
	}

	log.Info("Project has been pushed successfully")

	return nil
}

func (h *PutProject) createGerritProject(ctx context.Context, gitServer *codebaseApi.GitServer, privateSSHKey, projectName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating project in Gerrit")

	projectExist, err := h.gerrit.CheckProjectExist(gitServer.Spec.SshPort, privateSSHKey, gitServer.Spec.GitHost, gitServer.Spec.GitUser, projectName, log)
	if err != nil {
		return fmt.Errorf("failed to check if project exist in Gerrit: %w", err)
	}

	if projectExist {
		log.Info("Skip creating project in Gerrit, project already exist")
		return nil
	}

	err = h.gerrit.CreateProject(gitServer.Spec.SshPort, privateSSHKey, gitServer.Spec.GitHost, gitServer.Spec.GitUser, projectName, log)
	if err != nil {
		return fmt.Errorf("failed to create gerrit project: %w", err)
	}

	log.Info("Project created in Gerrit")

	return nil
}

func (h *PutProject) checkoutBranch(ctx context.Context, codebase *codebaseApi.Codebase, workDir string) error {
	log := ctrl.LoggerFrom(ctx).WithValues(
		"defaultBranch",
		codebase.Spec.DefaultBranch,
		"branchToCopy",
		codebase.Spec.BranchToCopyInDefaultBranch,
	)

	repoUrl, err := util.GetRepoUrl(codebase)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	if codebase.Spec.BranchToCopyInDefaultBranch != "" && codebase.Spec.DefaultBranch != codebase.Spec.BranchToCopyInDefaultBranch {
		log.Info("Start checkout branch to copy")

		err = CheckoutBranch(repoUrl, workDir, codebase.Spec.BranchToCopyInDefaultBranch, h.git, codebase, h.client)
		if err != nil {
			return fmt.Errorf("failed to checkout default branch %s: %w", codebase.Spec.DefaultBranch, err)
		}

		log.Info("Start replace default branch")

		err = h.replaceDefaultBranch(ctx, workDir, codebase.Spec.DefaultBranch, codebase.Spec.BranchToCopyInDefaultBranch)
		if err != nil {
			return fmt.Errorf("failed to replace master: %w", err)
		}

		return nil
	}

	log.Info("Start checkout branch")

	err = CheckoutBranch(repoUrl, workDir, codebase.Spec.DefaultBranch, h.git, codebase, h.client)
	if err != nil {
		return fmt.Errorf("failed to checkout default branch %s: %w", codebase.Spec.DefaultBranch, err)
	}

	log.Info("Checkout branch finished")

	return nil
}

func (h *PutProject) createGitThirdPartyProject(ctx context.Context, gitServer *codebaseApi.GitServer, gitProviderToken, projectName string) error {
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

		err := h.gerrit.SetHeadToBranch(
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
		return fmt.Errorf("failed to set default branch: %w", err)
	}

	log.Info("Default branch has been set")

	return nil
}

func (h *PutProject) tryToCloneRepo(ctx context.Context, repoUrl string, repositoryUsername, repositoryPassword *string, workDir string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("dest", workDir, "repoUrl", repoUrl)

	log.Info("Start cloning repository")

	if util.DoesDirectoryExist(workDir + "/.git") {
		log.Info("Repository already exists")

		return nil
	}

	if err := h.git.CloneRepository(repoUrl, repositoryUsername, repositoryPassword, workDir); err != nil {
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

	if err := h.git.Init(workDir); err != nil {
		return fmt.Errorf("failed to create git repository: %w", err)
	}

	if err := h.git.CommitChanges(workDir, "Initial commit"); err != nil {
		return fmt.Errorf("failed to commit all default content: %w", err)
	}

	log.Info("Commits have been squashed")

	return nil
}

func (h *PutProject) initialProjectProvisioning(ctx context.Context, codebase *codebaseApi.Codebase, wd string) error {
	if codebase.Spec.EmptyProject {
		return h.emptyProjectProvisioning(ctx, wd)
	}

	return h.notEmptyProjectProvisioning(ctx, codebase, wd)
}

func (h *PutProject) emptyProjectProvisioning(ctx context.Context, wd string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Initialing empty git repository")

	if err := h.git.Init(wd); err != nil {
		return fmt.Errorf("failed to create empty git repository: %w", err)
	}

	log.Info("Making initial commit")

	if err := h.git.CommitChanges(wd, "Initial commit", git.CommitAllowEmpty()); err != nil {
		return fmt.Errorf("failed to create Initial commit: %w", err)
	}

	return nil
}

func (h *PutProject) notEmptyProjectProvisioning(ctx context.Context, codebase *codebaseApi.Codebase, wd string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start initial provisioning for non-empty project")

	repoUrl, err := util.GetRepoUrl(codebase)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	repu, repp, err := GetRepositoryCredentialsIfExists(codebase, h.client)
	// we are ok if no credentials is found, assuming this is a public repo
	if err != nil && !k8sErrors.IsNotFound(err) {
		return fmt.Errorf("failed to get repository credentials: %w", err)
	}

	if !h.git.CheckPermissions(ctx, repoUrl, repu, repp) {
		return fmt.Errorf("failed to get access to the repository %v for user %v", repoUrl, *repu)
	}

	if err = h.tryToCloneRepo(ctx, repoUrl, repu, repp, wd); err != nil {
		return fmt.Errorf("failed to clone template project: %w", err)
	}

	if err = h.squashCommits(ctx, wd, codebase.Spec.Strategy); err != nil {
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
