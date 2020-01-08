package chain

import (
	"bytes"
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/codebase-operator/v2/pkg/vcs"
	"github.com/pkg/errors"
	"html/template"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"os"
	"time"
)

type PutProjectGerrit struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

const (
	GithubDomain = "https://github.com/epmd-edp"
)

func (h PutProjectGerrit) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start putting Codebase...")

	if c.Status.Status != model.StatusInit {
		rLog.Info("Codebase is not in initialized status. Skipped.", "name", c.Name,
			"status", c.Status.Status)
		return nil
	}
	log.Info("start handling codebase", "name", c.Name, "spec", c.Spec)

	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.AcceptCodebaseRegistration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.GerritRepositoryProvisioning); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	if err := util.CreateDirectory(wd); err != nil {
		return err
	}

	gs, us, err := util.GetConfigSettings(h.clientSet, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "unable get config settings")
	}

	if err := h.tryToCreateGerritPrivateKey(c.Namespace, wd); err != nil {
		setFailedFields(*c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "unable to write the Gerrit ssh key for %v codebase", c.Name)
	}

	if err := h.tryToCreateSshConfig(gs, us, wd, c.Namespace); err != nil {
		setFailedFields(*c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "creation of SSH config for %v codebase has been failed", c.Name)
	}

	ru, err := util.GetRepoUrl(GithubDomain, c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}
	rLog.Info("Repository URL to clone sources has been retrieved", "url", *ru)

	repu, repp, err := h.tryToGetRepositoryCredentials(c)
	if !git.CheckPermissions(*ru, repu, repp) {
		return fmt.Errorf("user %v cannot get access to the repository %v", repu, *ru)
	}

	if err := h.tryToCreateProjectInVcs(us, c.Name, c.Namespace); err != nil {
		setFailedFields(*c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "unable to create project in VCS")
	}

	if err := h.tryToCloneRepo(*ru, repu, repp, wd, c.Name); err != nil {
		setFailedFields(*c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrap(err, "clonning project hsa been failed")
	}

	if err := h.tryToPushProjectToGerrit(gs, c.Name, wd, c.Namespace); err != nil {
		setFailedFields(*c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "push to gerrit for codebase %v has been failed", c.Name)
	}
	rLog.Info("end creating project in Gerrit")
	return nextServeOrNil(h.next, c)
}

func (h PutProjectGerrit) tryToPushProjectToGerrit(gs *model.GerritSettings, codebaseName, workDir, namespace string) error {
	gf := &model.GerritConf{
		GerritKeyPath: fmt.Sprintf("%v/gerrit-private.key", workDir),
		GerritHost:    fmt.Sprintf("gerrit.%v", namespace),
		SshPort:       gs.SshPort,
		WorkDir:       workDir,
	}
	if err := h.createProjectInGerrit(gf, codebaseName); err != nil {
		return errors.Wrapf(err, "creation project in Gerrit for codebase %v has been failed", codebaseName)
	}
	return h.pushToGerrit(gf, codebaseName)
}

func (h PutProjectGerrit) tryToCreateGerritPrivateKey(namespace, workDir string) error {
	idrsa, _, err := h.getGerritCredentials(namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get Gerrit credentials")
	}
	return h.createGerritPrivateKey(idrsa, workDir)
}

func (h PutProjectGerrit) tryToCreateSshConfig(gs *model.GerritSettings, us *model.UserSettings, workDir, namespace string) error {
	sshConf, err := h.getSshConfig(us, gs, namespace, workDir)
	if err != nil {
		return errors.Wrap(err, "couldn't create SSH config model")
	}
	return h.createSshConfig(sshConf)
}

func (h PutProjectGerrit) pushToGerrit(conf *model.GerritConf, codebaseName string) error {
	log.Info("Start pushing project to Gerrit ", "codebase name", codebaseName)
	rp := fmt.Sprintf("%v/%v", conf.WorkDir, codebaseName)
	if err := gerrit.AddRemoteLinkToGerrit(rp, conf.GerritHost, conf.SshPort, codebaseName); err != nil {
		return errors.Wrap(err, "couldn't add remote link to Gerrit")
	}

	k, err := readFile(conf.GerritKeyPath)
	if err != nil {
		return err
	}

	if err := git.PushChanges(k, "project-creator", rp); err != nil {
		if err.Error() == "already up-to-date" {
			log.Info("project already up-to-date. skip pushing")
			return nil
		}
		return err
	}

	return nil
}

func readFile(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	log.Info("Private key has been read")
	return string(b), nil
}

func (h PutProjectGerrit) createProjectInGerrit(conf *model.GerritConf, codebaseName string) error {
	log.Info("Start creating project in Gerrit", "codebase name")
	projectExist, err := gerrit.CheckProjectExist(conf.GerritKeyPath, conf.GerritHost, conf.SshPort, codebaseName)
	if err != nil {
		return errors.Wrap(err, "couldn't check project")
	}
	if *projectExist {
		log.Info("couldn't create project in Gerrit. Project already exists", "name", codebaseName)
		return nil
	}

	if err := gerrit.CreateProject(conf.GerritKeyPath, conf.GerritHost, conf.SshPort, codebaseName); err != nil {
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

func (h PutProjectGerrit) createSshConfig(ssh *model.SshConfig) error {
	sshPath := "/home/codebase-operator/.ssh"
	log.Info("start creation of SSH config", "path", sshPath)
	var config bytes.Buffer
	if _, err := os.Stat(sshPath); os.IsNotExist(err) {
		if err := os.MkdirAll(sshPath, 0744); err != nil {
			return err
		}
	}

	tmpl, err := template.
		New("config.tmpl").
		ParseFiles("/usr/local/bin/templates/ssh/config.tmpl")
	if err != nil {
		return err
	}

	if err := tmpl.Execute(&config, ssh); err != nil {
		return err
	}

	f, err := os.OpenFile(sshPath+"/config", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, config.String())
	if err != nil {
		return err
	}

	return nil
}

func (h PutProjectGerrit) getSshConfig(us *model.UserSettings, gs *model.GerritSettings, namespace, workDir string) (*model.SshConfig, error) {
	log.Info("creating SSH config model")
	vcsGroup, err := url.Parse(us.VcsGroupNameUrl)
	if err != nil {
		return nil, err
	}

	return &model.SshConfig{
		CicdNamespace:         namespace,
		SshPort:               gs.SshPort,
		GerritKeyPath:         fmt.Sprintf("%v/gerrit-private.key", workDir),
		VcsIntegrationEnabled: us.VcsIntegrationEnabled,
		ProjectVcsHostname:    vcsGroup.Host,
		VcsSshPort:            us.VcsSshPort,
		VcsKeyPath:            workDir + "/vcs-private.key",
	}, nil
}

func (h PutProjectGerrit) createGerritPrivateKey(privateKey string, workDir string) error {
	log.Info("start creation of Gerrit private key")
	path := fmt.Sprintf("%v/gerrit-private.key", workDir)
	if err := ioutil.WriteFile(path, []byte(privateKey), 400); err != nil {
		if os.IsExist(err) {
			log.Info("file already exists", "path", path)
			return nil
		}
		return err
	}
	return nil
}

func (h PutProjectGerrit) getGerritCredentials(namespace string) (string, string, error) {
	log.Info("getting Gerrit credentials", "namespace", namespace)
	s, err := h.clientSet.CoreClient.
		Secrets(namespace).
		Get("gerrit-project-creator", metav1.GetOptions{})

	if err != nil {
		return "", "", err
	}
	return string(s.Data[util.PrivateSShKeyName]), string(s.Data["id_rsa.pub"]), nil
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
	}

	if err := h.clientSet.Client.Status().Update(context.TODO(), c); err != nil {
		if err := h.clientSet.Client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}

func setFailedFields(c edpv1alpha1.Codebase, a edpv1alpha1.ActionType, message string) {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusFailed,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          a,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}
}
