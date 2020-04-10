package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/codebase-operator/v2/pkg/vcs"
	"github.com/pkg/errors"
	"time"
)

type PutProjectGerrit struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
	cr        repository.CodebaseRepository
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

	gs, us, err := util.GetConfigSettings(h.clientSet.CoreClient, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable get config settings")
	}

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "couldn't build repo url")
	}
	rLog.Info("Repository URL to clone sources has been retrieved", "url", *ru)

	edpN, err := helper.GetEDPName(h.clientSet.Client, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(c.Name, *edpN)
	if err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "couldn't get pushed value for %v codebase", c.Name)
	}

	if *ps == util.ProjectPushedStatus || *ps == util.ProjectTemplatesPushedStatus {
		log.V(2).Info("skip pushing to gerrit. project already pushed", "name", c.Name)
		return nextServeOrNil(h.next, c)
	}

	repu, repp, err := h.tryToGetRepositoryCredentials(c)
	if !git.CheckPermissions(*ru, repu, repp) {
		msg := fmt.Errorf("user %v cannot get access to the repository %v", repu, *ru)
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, msg.Error())
		return msg
	}

	if err := h.tryToCreateProjectInVcs(us, c.Name, c.Namespace); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable to create project in VCS")
	}

	if err := h.tryToCloneRepo(*ru, repu, repp, wd, c.Name); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "cloning project hsa been failed")
	}

	if err := h.tryToPushProjectToGerrit(gs.SshPort, c.Name, wd, c.Namespace); err != nil {
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

func (h PutProjectGerrit) tryToPushProjectToGerrit(sshPort int32, codebaseName, workDir, namespace string) error {
	s, err := util.GetSecret(*h.clientSet.CoreClient, "gerrit-project-creator", namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	host := fmt.Sprintf("gerrit.%v", namespace)
	if err := h.tryToCreateProjectInGerrit(sshPort, idrsa, host, codebaseName); err != nil {
		return errors.Wrapf(err, "creation project in Gerrit for codebase %v has been failed", codebaseName)
	}

	d := fmt.Sprintf("%v/%v", workDir, codebaseName)
	if err := h.pushToGerrit(sshPort, idrsa, host, codebaseName, d); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) pushToGerrit(sshPost int32, idrsa, host, codebaseName, directory string) error {
	log.Info("Start pushing project to Gerrit ", "codebase name", codebaseName)
	if err := gerrit.AddRemoteLinkToGerrit(directory, host, sshPost, codebaseName); err != nil {
		return errors.Wrap(err, "couldn't add remote link to Gerrit")
	}
	if err := git.PushChanges(idrsa, "project-creator", directory); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) tryToCreateProjectInGerrit(sshPort int32, idrsa, host, codebaseName string) error {
	log.Info("Start creating project in Gerrit", "codebase name")
	projectExist, err := gerrit.CheckProjectExist(sshPort, idrsa, host, codebaseName)
	if err != nil {
		return errors.Wrap(err, "couldn't check project")
	}
	if *projectExist {
		log.Info("couldn't create project in Gerrit. Project already exists", "name", codebaseName)
		return nil
	}

	if err := gerrit.CreateProject(sshPort, idrsa, host, codebaseName); err != nil {
		return err
	}
	return nil
}

func (h PutProjectGerrit) tryToCloneRepo(repoUrl string, repositoryUsername string, repositoryPassword, workDir, codebaseName string) error {
	destination := fmt.Sprintf("%v/%v", workDir, codebaseName)
	log.Info("Start cloning repository", "src", repoUrl, "dest", destination)

	if util.DoesDirectoryExist(destination) {
		log.Info("project already exists", "name", codebaseName)
		return nil
	}

	if err := git.CloneRepository(repoUrl, repositoryUsername, repositoryPassword, destination); err != nil {
		return err
	}
	log.Info("Repository has been cloned", "src", repoUrl, "dest", destination)
	return nil
}

func (h PutProjectGerrit) tryToCreateProjectInVcs(us *model.UserSettings, codebaseName, namespace string) error {
	log.Info("Start creation project in VCS", "codebase name", codebaseName)
	if us.VcsIntegrationEnabled {
		if err := vcs.СreateProjectInVcs(*h.clientSet.CoreClient, us, codebaseName, namespace); err != nil {
			return err
		}
		return nil
	}
	log.Info("VCS integration isn't enabled. Skip creation project in VCS")
	return nil
}

func (h PutProjectGerrit) tryToGetRepositoryCredentials(c *v1alpha1.Codebase) (string, string, error) {
	if c.Spec.Repository != nil {
		return h.getRepoCreds(c.Name, c.Namespace)
	}
	return "", "", nil
}

func (h PutProjectGerrit) getRepoCreds(codebaseName, namespace string) (string, string, error) {
	secret := fmt.Sprintf("repository-codebase-%v-temp", codebaseName)
	repositoryUsername, repositoryPassword, err := util.GetVcsBasicAuthConfig(*h.clientSet.CoreClient, namespace, secret)
	if err != nil {
		return "", "", err
	}
	return repositoryUsername, repositoryPassword, nil
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
	}

	if err := h.clientSet.Client.Status().Update(context.TODO(), c); err != nil {
		if err := h.clientSet.Client.Update(context.TODO(), c); err != nil {
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
	}
}
