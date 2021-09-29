package chain

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
	"github.com/pkg/errors"
)

type PutProjectGerrit struct {
	next   handler.CodebaseHandler
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func (h PutProjectGerrit) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start putting Codebase...")
	rLog.Info("codebase data", "spec", c.Spec)
	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.AcceptCodebaseRegistration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.GerritRepositoryProvisioning); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/templates", c.Namespace, c.Name)
	if err := util.CreateDirectory(wd); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return err
	}

	port, err := util.GetGerritPort(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable get gerrit port")
	}

	us, err := util.GetUserSettings(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable get user settings settings")
	}

	edpN, err := helper.GetEDPName(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(c.Name, *edpN)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "couldn't get project_status value for %v codebase", c.Name)
	}

	var status = []string{util.ProjectPushedStatus, util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}
	if util.ContainsString(status, *ps) {
		log.V(2).Info("skip pushing to gerrit. project already pushed", "name", c.Name)
		return nextServeOrNil(h.next, c)
	}

	if err := h.tryToCreateProjectInVcs(us, c.Name, c.Namespace); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable to create project in VCS")
	}

	if err := h.initialProjectProvisioning(c, rLog, wd); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "initial provisioning of codebase %v has been failed", c.Name)
	}

	if err := h.tryToPushProjectToGerrit(c, *port, c.Name, wd, c.Namespace, c.Spec.DefaultBranch, c.Spec.Strategy); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "push to gerrit for codebase %v has been failed", c.Name)
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectPushedStatus, c.Name, *edpN); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	rLog.Info("end creating project in Gerrit")
	return nextServeOrNil(h.next, c)
}

func (h PutProjectGerrit) tryToPushProjectToGerrit(c *v1alpha1.Codebase, sshPort int32, codebaseName, workDir, namespace, branchName string, strategy v1alpha1.Strategy) error {
	s, err := util.GetSecret(h.client, "gerrit-project-creator", namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	host := fmt.Sprintf("gerrit.%v", namespace)
	if err := h.tryToCreateProjectInGerrit(sshPort, idrsa, host, codebaseName); err != nil {
		return errors.Wrapf(err, "creation project in Gerrit for codebase %v has been failed", codebaseName)
	}

	d := fmt.Sprintf("%v/%v", workDir, codebaseName)

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	if err := CheckoutBranch(ru, d, branchName, h.git, c, h.client); err != nil {
		return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", branchName)
	}

	if err := h.pushToGerrit(sshPort, idrsa, host, codebaseName, d, strategy); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) pushToGerrit(sshPost int32, idrsa, host, codebaseName, directory string, strategy v1alpha1.Strategy) error {
	log.Info("Start pushing project to Gerrit ", "codebase name", codebaseName)
	if err := gerrit.AddRemoteLinkToGerrit(directory, host, sshPost, codebaseName, log); err != nil {
		return errors.Wrap(err, "couldn't add remote link to Gerrit")
	}
	if err := h.git.PushChanges(idrsa, "project-creator", directory); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) tryToCreateProjectInGerrit(sshPort int32, idrsa, host, codebaseName string) error {
	log.Info("Start creating project in Gerrit", "codebase name", codebaseName)
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
	destination := fmt.Sprintf("%v/%v", workDir, codebaseName)
	log.Info("Start cloning repository", "src", repoUrl, "dest", destination)

	if util.DoesDirectoryExist(destination) {
		log.Info("project already exists", "name", codebaseName)
		return nil
	}

	if err := h.git.CloneRepository(repoUrl, repositoryUsername, repositoryPassword, destination); err != nil {
		return err
	}
	log.Info("Repository has been cloned", "src", repoUrl, "dest", destination)
	return nil
}

func (h PutProjectGerrit) tryToCreateProjectInVcs(us *model.UserSettings, codebaseName, namespace string) error {
	log.Info("Start creation project in VCS", "codebase name", codebaseName)
	if us.VcsIntegrationEnabled {
		if err := vcs.СreateProjectInVcs(h.client, us, codebaseName, namespace); err != nil {
			return err
		}
		return nil
	}
	log.Info("VCS integration isn't enabled. Skip creation project in VCS")
	return nil
}

func (h PutProjectGerrit) setIntermediateSuccessFields(c *edpv1alpha1.Codebase, action edpv1alpha1.ActionType) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Action:          action,
		Result:          edpv1alpha1.Success,
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

func setFailedFields(c *edpv1alpha1.Codebase, a edpv1alpha1.ActionType, message string) {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusFailed,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          a,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
		FailureCount:    c.Status.FailureCount,
		Git:             c.Status.Git,
	}
}

func (h PutProjectGerrit) tryToSquashCommits(workDir, codebaseName string, strategy v1alpha1.Strategy) error {
	if strategy != v1alpha1.Create {
		return nil
	}
	destination := fmt.Sprintf("%v/%v", workDir, codebaseName)

	err := os.RemoveAll(destination + "/.git")
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while removing .git folder")
	}

	if err := h.git.Init(destination); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating git repository")
	}

	if err := h.git.CommitChanges(destination, "Init commit"); err != nil {
		return errors.Wrapf(err, "an error has occurred while committing all default content")
	}
	return nil
}

func (h PutProjectGerrit) initialProjectProvisioning(c *v1alpha1.Codebase, rLog logr.Logger, wd string) error {
	if c.Spec.EmptyProject {
		return h.emptyProjectProvisioning(wd, c.Name)
	}
	return h.notEmptyProjectProvisioning(c, rLog, wd)
}

func (h PutProjectGerrit) emptyProjectProvisioning(wd, codebaseName string) error {
	log.Info("Start initial provisioning for empty project", "codebase name", codebaseName)
	destination := fmt.Sprintf("%v/%v", wd, codebaseName)
	if err := h.git.Init(destination); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating empty git repository")
	}

	if err := h.git.CommitChanges(destination, "Init commit"); err != nil {
		return errors.Wrapf(err, "an error has occurred while committing empty default content")
	}

	return nil
}

func (h PutProjectGerrit) notEmptyProjectProvisioning(c *v1alpha1.Codebase, rLog logr.Logger, wd string) error {
	log.Info("Start initial provisioning for not empty project", "codebase name", c.Name)
	ru, err := util.GetRepoUrl(c)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "couldn't build repo url")
	}
	rLog.Info("Repository URL to clone sources has been retrieved", "url", *ru)

	repu, repp, err := GetRepositoryCredentialsIfExists(c, h.client)

	if !h.git.CheckPermissions(*ru, repu, repp) {
		msg := fmt.Errorf("user %v cannot get access to the repository %v", repu, *ru)
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, msg.Error())
		return msg
	}

	if err := h.tryToCloneRepo(*ru, repu, repp, wd, c.Name); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "cloning project hsa been failed")
	}

	if err := h.tryToSquashCommits(wd, c.Name, c.Spec.Strategy); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "squash commits been failed")
	}
	return nil
}
