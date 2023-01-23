package chain

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
)

const (
	logCodebaseNameKey = "codebase_name"
)

type PutProjectGerrit struct {
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func NewPutProjectGerrit(c client.Client, cr repository.CodebaseRepository, g git.Git) *PutProjectGerrit {
	return &PutProjectGerrit{client: c, cr: cr, git: g}
}

func (h *PutProjectGerrit) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues(logCodebaseNameKey, c.Name)
	rLog.Info("Start putting Codebase...")
	rLog.Info("codebase data", "spec", c.Spec)

	edpN, err := helper.GetEDPName(ctx, h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())

		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(ctx, c.Name, *edpN)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())

		return errors.Wrapf(err, "couldn't get project_status value for %v codebase", c.Name)
	}

	var status = []string{util.ProjectPushedStatus, util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}
	if util.ContainsString(status, ps) {
		log.Info("skip pushing to gerrit. project already pushed", "name", c.Name)

		return nil
	}

	err = setIntermediateSuccessFields(ctx, h.client, c, codebaseApi.GerritRepositoryProvisioning)
	if err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)

	err = util.CreateDirectory(wd)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())

		return fmt.Errorf("failed to create dir %q: %w", wd, err)
	}

	port, err := util.GetGerritPort(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())

		return errors.Wrap(err, "unable get gerrit port")
	}

	us, err := util.GetUserSettings(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable get user settings settings")
	}

	err = h.tryToCreateProjectInVcs(us, c.Name, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable to create project in VCS")
	}

	err = h.initialProjectProvisioning(c, rLog, wd)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "initial provisioning of codebase %v has been failed", c.Name)
	}

	err = h.tryToPushProjectToGerrit(c, *port, c.Name, wd, c.Namespace, c.Spec.DefaultBranch)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "push to gerrit for codebase %v has been failed", c.Name)
	}

	err = h.cr.UpdateProjectStatusValue(ctx, util.ProjectPushedStatus, c.Name, *edpN)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	rLog.Info("end creating project in Gerrit")

	return nil
}

func (h *PutProjectGerrit) tryToPushProjectToGerrit(c *codebaseApi.Codebase, sshPort int32, codebaseName, workDir,
	namespace, branchName string) error {
	s, err := util.GetSecret(h.client, "gerrit-project-creator", namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	host := fmt.Sprintf("gerrit.%v", namespace)

	err = h.tryToCreateProjectInGerrit(sshPort, idrsa, host, codebaseName)
	if err != nil {
		return errors.Wrapf(err, "creation project in Gerrit for codebase %v has been failed", codebaseName)
	}

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	if c.Spec.BranchToCopyInDefaultBranch != "" && c.Spec.DefaultBranch != c.Spec.BranchToCopyInDefaultBranch {
		err = CheckoutBranch(ru, workDir, c.Spec.BranchToCopyInDefaultBranch, h.git, c, h.client)
		if err != nil {
			return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", branchName)
		}

		err = h.replaceDefaultBranch(workDir, c.Spec.DefaultBranch, c.Spec.BranchToCopyInDefaultBranch)
		if err != nil {
			return errors.Wrap(err, "unable to replace master")
		}
	} else {
		err = CheckoutBranch(ru, workDir, branchName, h.git, c, h.client)
		if err != nil {
			return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", branchName)
		}
	}

	err = h.pushToGerrit(sshPort, idrsa, host, codebaseName, workDir)
	if err != nil {
		return err
	}

	// set remote head to default branch
	err = gerrit.SetHeadToBranch(sshPort, idrsa, host, codebaseName, branchName, log)
	if err != nil {
		return fmt.Errorf("set remote HEAD for codebase %v to default branch %v has been failed: %w", codebaseName, branchName, err)
	}

	return nil
}

func (h *PutProjectGerrit) replaceDefaultBranch(directory, defaultBranchName, newBranchName string) error {
	log.Info("Replace master branch with %s")

	if err := h.git.RemoveBranch(directory, defaultBranchName); err != nil {
		return errors.Wrap(err, "unable to remove master branch")
	}

	if err := h.git.CreateChildBranch(directory, newBranchName, defaultBranchName); err != nil {
		return errors.Wrap(err, "unable to create child branch")
	}

	return nil
}

func (h *PutProjectGerrit) pushToGerrit(sshPort int32, idrsa, host, codebaseName, directory string) error {
	log.Info("Start pushing project to Gerrit ", logCodebaseNameKey, codebaseName)

	if err := gerrit.AddRemoteLinkToGerrit(directory, host, sshPort, codebaseName, log); err != nil {
		return errors.Wrap(err, "couldn't add remote link to Gerrit")
	}

	// push branches
	if err := h.git.PushChanges(idrsa, "project-creator", directory, sshPort, "--all"); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	// push tags as well
	if err := h.git.PushChanges(idrsa, "project-creator", directory, sshPort, "--tags"); err != nil {
		return fmt.Errorf("failed to push changes into git: %w", err)
	}

	return nil
}

func (*PutProjectGerrit) tryToCreateProjectInGerrit(sshPort int32, idrsa, host, codebaseName string) error {
	log.Info("Start creating project in Gerrit", logCodebaseNameKey, codebaseName)

	projectExist, err := gerrit.CheckProjectExist(sshPort, idrsa, host, codebaseName, log)
	if err != nil {
		return errors.Wrap(err, "couldn't versionFileExists project")
	}

	if *projectExist {
		log.Info("couldn't create project in Gerrit. Project already exists", "name", codebaseName)
		return nil
	}

	err = gerrit.CreateProject(sshPort, idrsa, host, codebaseName, log)
	if err != nil {
		return fmt.Errorf("failed to create gerrit project: %w", err)
	}

	return nil
}

func (h *PutProjectGerrit) tryToCloneRepo(repoUrl string, repositoryUsername, repositoryPassword *string, workDir, codebaseName string) error {
	log.Info("Start cloning repository", "src", repoUrl, "dest", workDir)

	if util.DoesDirectoryExist(workDir + "/.git") {
		log.Info("repository already exists", logCodebaseNameKey, codebaseName)

		return nil
	}

	if err := h.git.CloneRepository(repoUrl, repositoryUsername, repositoryPassword, workDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info("Repository has been cloned", "src", repoUrl, "dest", workDir)

	return nil
}

func (h *PutProjectGerrit) tryToCreateProjectInVcs(us *model.UserSettings, codebaseName, namespace string) error {
	log.Info("Start project creation in VCS", logCodebaseNameKey, codebaseName)

	if !us.VcsIntegrationEnabled {
		log.Info("VCS integration isn't enabled. Skip creation project in VCS")

		return nil
	}

	err := vcs.CreateProjectInVcs(h.client, us, codebaseName, namespace)
	if err != nil {
		return fmt.Errorf("failed to craete a project in VCS: %w", err)
	}

	return nil
}

func (h *PutProjectGerrit) tryToSquashCommits(workDir, codebaseName string, strategy codebaseApi.Strategy) error {
	if strategy != codebaseApi.Create {
		return nil
	}

	log.Info("Start squashing commits", logCodebaseNameKey, codebaseName)

	err := os.RemoveAll(workDir + "/.git")
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while removing .git folder")
	}

	if err := h.git.Init(workDir); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating git repository")
	}

	if err := h.git.CommitChanges(workDir, "Initial commit"); err != nil {
		return errors.Wrapf(err, "an error has occurred while committing all default content")
	}

	return nil
}

func (h *PutProjectGerrit) initialProjectProvisioning(c *codebaseApi.Codebase, rLog logr.Logger, wd string) error {
	if c.Spec.EmptyProject {
		return h.emptyProjectProvisioning(wd, c.Name)
	}

	return h.notEmptyProjectProvisioning(c, rLog, wd)
}

func (h *PutProjectGerrit) emptyProjectProvisioning(wd, codebaseName string) error {
	log.Info("Start initial provisioning for empty project", logCodebaseNameKey, codebaseName)

	if err := h.git.Init(wd); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating empty git repository")
	}

	if err := h.git.CommitChanges(wd, "Initial commit"); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating Initial commit")
	}

	return nil
}

func (h *PutProjectGerrit) notEmptyProjectProvisioning(c *codebaseApi.Codebase, rLog logr.Logger, wd string) error {
	log.Info("Start initial provisioning for non-empty project", logCodebaseNameKey, c.Name)

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "couldn't build repo url")
	}

	rLog.Info("Repository URL with template has been retrieved", "url", *ru)

	repu, repp, err := GetRepositoryCredentialsIfExists(c, h.client)
	// we are ok if no credentials is found, assuming this is a public repo
	if err != nil && !k8sErrors.IsNotFound(err) {
		return errors.Wrap(err, "Unable to get repository credentials")
	}

	if !h.git.CheckPermissions(*ru, repu, repp) {
		msg := fmt.Errorf("user %v cannot get access to the repository %v", *repu, *ru)
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, msg.Error())

		return msg
	}

	if err := h.tryToCloneRepo(*ru, repu, repp, wd, c.Name); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "cloning template project has been failed")
	}

	if err := h.tryToSquashCommits(wd, c.Name, c.Spec.Strategy); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "squash commits in a template repo has been failed")
	}

	return nil
}

func setFailedFields(c *codebaseApi.Codebase, a codebaseApi.ActionType, message string) {
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
	}
}