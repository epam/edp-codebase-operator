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

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
)

type PutProjectGerrit struct {
	next   handler.CodebaseHandler
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func (h PutProjectGerrit) ServeRequest(c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("Start putting Codebase...")
	rLog.Info("codebase data", "spec", c.Spec)

	edpN, err := helper.GetEDPName(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(c.Name, *edpN)
	if err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "couldn't get project_status value for %v codebase", c.Name)
	}

	var status = []string{util.ProjectPushedStatus, util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}
	if util.ContainsString(status, *ps) {
		log.Info("skip pushing to gerrit. project already pushed", "name", c.Name)
		return nextServeOrNil(h.next, c)
	}

	if err := h.setIntermediateSuccessFields(c, codebaseApi.GerritRepositoryProvisioning); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	if err := util.CreateDirectory(wd); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return err
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

	if err := h.tryToCreateProjectInVcs(us, c.Name, c.Namespace); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable to create project in VCS")
	}

	if err := h.initialProjectProvisioning(c, rLog, wd); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "initial provisioning of codebase %v has been failed", c.Name)
	}

	if err := h.tryToPushProjectToGerrit(c, *port, c.Name, wd, c.Namespace, c.Spec.DefaultBranch, c.Spec.Strategy); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "push to gerrit for codebase %v has been failed", c.Name)
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectPushedStatus, c.Name, *edpN); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	rLog.Info("end creating project in Gerrit")
	return nextServeOrNil(h.next, c)
}

func (h PutProjectGerrit) tryToPushProjectToGerrit(c *codebaseApi.Codebase, sshPort int32, codebaseName, workDir,
	namespace, branchName string, strategy codebaseApi.Strategy) error {
	s, err := util.GetSecret(h.client, "gerrit-project-creator", namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	host := fmt.Sprintf("gerrit.%v", namespace)
	if err := h.tryToCreateProjectInGerrit(sshPort, idrsa, host, codebaseName); err != nil {
		return errors.Wrapf(err, "creation project in Gerrit for codebase %v has been failed", codebaseName)
	}

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	if c.Spec.BranchToCopyInDefaultBranch != "" && c.Spec.DefaultBranch != c.Spec.BranchToCopyInDefaultBranch {
		if err := CheckoutBranch(ru, workDir, c.Spec.BranchToCopyInDefaultBranch, h.git, c, h.client); err != nil {
			return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", branchName)
		}

		if err := h.replaceDefaultBranch(workDir, c.Spec.DefaultBranch, c.Spec.BranchToCopyInDefaultBranch); err != nil {
			return errors.Wrap(err, "unable to replace master")
		}
	} else {
		if err := CheckoutBranch(ru, workDir, branchName, h.git, c, h.client); err != nil {
			return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", branchName)
		}
	}

	if err := h.pushToGerrit(sshPort, idrsa, host, codebaseName, workDir, strategy); err != nil {
		return err
	}
	// set remote head to default branch
	if err := gerrit.SetHeadToBranch(sshPort, idrsa, host, codebaseName, branchName, log); err != nil {
		return fmt.Errorf("set remote HEAD for codebase %v to default branch %v has been failed: %w", codebaseName, branchName, err)
	}
	return nil
}

func (h PutProjectGerrit) replaceDefaultBranch(directory, defaultBranchName, newBranchName string) error {
	log.Info("Replace master branch with %s")

	if err := h.git.RemoveBranch(directory, defaultBranchName); err != nil {
		return errors.Wrap(err, "unable to remove master branch")
	}

	if err := h.git.CreateChildBranch(directory, newBranchName, defaultBranchName); err != nil {
		return errors.Wrap(err, "unable to create child branch")
	}

	return nil
}

func (h PutProjectGerrit) pushToGerrit(sshPost int32, idrsa, host, codebaseName, directory string, strategy codebaseApi.Strategy) error {
	log.Info("Start pushing project to Gerrit ", "codebase_name", codebaseName)
	if err := gerrit.AddRemoteLinkToGerrit(directory, host, sshPost, codebaseName, log); err != nil {
		return errors.Wrap(err, "couldn't add remote link to Gerrit")
	}
	// push branches
	if err := h.git.PushChanges(idrsa, "project-creator", directory, "--all"); err != nil {
		return err
	}
	// push tags as well
	if err := h.git.PushChanges(idrsa, "project-creator", directory, "--tags"); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) tryToCreateProjectInGerrit(sshPort int32, idrsa, host, codebaseName string) error {
	log.Info("Start creating project in Gerrit", "codebase_name", codebaseName)
	projectExist, err := gerrit.CheckProjectExist(sshPort, idrsa, host, codebaseName, log)
	if err != nil {
		return errors.Wrap(err, "couldn't versionFileExists project")
	}
	if *projectExist {
		log.Info("couldn't create project in Gerrit. Project already exists", "name", codebaseName)
		return nil
	}

	if err := gerrit.CreateProject(sshPort, idrsa, host, codebaseName, log); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) tryToCloneRepo(repoUrl string, repositoryUsername *string, repositoryPassword *string, workDir, codebaseName string) error {

	log.Info("Start cloning repository", "src", repoUrl, "dest", workDir)

	if util.DoesDirectoryExist(workDir + "/.git") {
		log.Info("repository already exists", "codebase_name", codebaseName)
		return nil
	}

	if err := h.git.CloneRepository(repoUrl, repositoryUsername, repositoryPassword, workDir); err != nil {
		return err
	}
	log.Info("Repository has been cloned", "src", repoUrl, "dest", workDir)
	return nil
}

func (h PutProjectGerrit) tryToCreateProjectInVcs(us *model.UserSettings, codebaseName, namespace string) error {
	log.Info("Start project creation in VCS", "codebase_name", codebaseName)
	if us.VcsIntegrationEnabled {
		if err := vcs.СreateProjectInVcs(h.client, us, codebaseName, namespace); err != nil {
			return err
		}
		return nil
	}
	log.Info("VCS integration isn't enabled. Skip creation project in VCS")
	return nil
}

func (h PutProjectGerrit) setIntermediateSuccessFields(c *codebaseApi.Codebase, action codebaseApi.ActionType) error {
	c.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Action:          action,
		Result:          codebaseApi.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    c.Status.FailureCount,
		Git:             c.Status.Git,
	}

	if err := h.client.Status().Update(context.TODO(), c); err != nil {
		if err := h.client.Update(context.TODO(), c); err != nil {
			return err
		}
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
	}
}

func (h PutProjectGerrit) tryToSquashCommits(workDir, codebaseName string, strategy codebaseApi.Strategy) error {
	if strategy != codebaseApi.Create {
		return nil
	}

	log.Info("Start squashing commits", "codebase_name", codebaseName)
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

func (h PutProjectGerrit) initialProjectProvisioning(c *codebaseApi.Codebase, rLog logr.Logger, wd string) error {
	if c.Spec.EmptyProject {
		return h.emptyProjectProvisioning(wd, c.Name)
	}
	return h.notEmptyProjectProvisioning(c, rLog, wd)
}

func (h PutProjectGerrit) emptyProjectProvisioning(wd, codebaseName string) error {
	log.Info("Start initial provisioning for empty project", "codebase_name", codebaseName)

	if err := h.git.Init(wd); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating empty git repository")
	}

	if err := h.git.CommitChanges(wd, "Initial commit"); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating Initial commit")
	}

	return nil
}

func (h PutProjectGerrit) notEmptyProjectProvisioning(c *codebaseApi.Codebase, rLog logr.Logger, wd string) error {
	log.Info("Start initial provisioning for non-empty project", "codebase_name", c.Name)
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
