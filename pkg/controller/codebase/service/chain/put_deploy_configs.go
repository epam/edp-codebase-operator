package chain

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/template"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

type PutDeployConfigs struct {
	next   handler.CodebaseHandler
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func (h PutDeployConfigs) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start pushing configs...")

	port, err := util.GetGerritPort(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrap(err, "unable get gerrit port")
	}

	if err := h.tryToPushConfigs(*c, *port); err != nil {
		setFailedFields(c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs for %v codebase", c.Name)
	}
	rLog.Info("end pushing configs")
	return nextServeOrNil(h.next, c)
}

func (h PutDeployConfigs) tryToPushConfigs(c edpv1alpha1.Codebase, sshPort int32) error {
	edpN, err := helper.GetEDPName(h.client, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(c.Name, *edpN)
	if err != nil {
		return errors.Wrapf(err, "couldn't get project_status value for %v codebase", c.Name)
	}

	var status = []string{util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}
	if util.ContainsString(status, *ps) {
		log.V(2).Info("skip pushing templates to gerrit. templates already pushed", "name", c.Name)
		return nil
	}

	s, err := util.GetSecret(h.client, "gerrit-project-creator", c.Namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}
	idrsa := string(s.Data[util.PrivateSShKeyName])

	u := "project-creator"
	url := fmt.Sprintf("ssh://gerrit.%v:%v/%v", c.Namespace, sshPort, c.Name)
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	td := fmt.Sprintf("%v/%v", wd, "templates")
	d := fmt.Sprintf("%v/%v", td, c.Name)

	if !util.DoesDirectoryExist(d) || util.IsDirectoryEmpty(d) {
		if err := h.cloneProjectRepoFromGerrit(sshPort, idrsa, c.Name, c.Namespace, url, td); err != nil {
			return err
		}
	}

	ru, err := util.GetRepoUrl(&c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	if err := CheckoutBranch(ru, d, c.Spec.DefaultBranch, h.git, &c, h.client); err != nil {
		return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", c.Spec.DefaultBranch)
	}

	if err := template.PrepareTemplates(h.client, c); err != nil {
		return err
	}

	if err := h.git.CommitChanges(d, fmt.Sprintf("Add template for %v", c.Name)); err != nil {
		return err
	}

	if err := h.git.PushChanges(idrsa, u, d); err != nil {
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectTemplatesPushedStatus, c.Name, *edpN); err != nil {
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	return nil
}

func (h PutDeployConfigs) cloneProjectRepoFromGerrit(sshPort int32, idrsa, name, namespace, cloneSshUrl, td string) error {
	log.Info("start cloning repository from Gerrit", "ssh url", cloneSshUrl)

	var (
		s *ssh.Session
		c *ssh.Client

		host = fmt.Sprintf("gerrit.%v", namespace)
	)

	sshcl, err := gerrit.SshInit(sshPort, idrsa, host, log)
	if err != nil {
		return errors.Wrap(err, "couldn't initialize SSH client")
	}

	if s, c, err = sshcl.NewSession(); err != nil {
		return errors.Wrap(err, "couldn't initialize SSH session")
	}

	defer func() {
		if deferErr := s.Close(); deferErr != nil {
			err = deferErr
		}
		if deferErr := c.Close(); deferErr != nil {
			err = deferErr
		}
	}()

	d := fmt.Sprintf("%v/%v", td, name)
	if err := h.git.CloneRepositoryBySsh(idrsa, "project-creator", cloneSshUrl, d); err != nil {
		return err
	}

	destinationPath := fmt.Sprintf("%v/%v/.git/hooks", td, name)
	if err := util.CreateDirectory(destinationPath); err != nil {
		return errors.Wrapf(err, "couldn't create folder %v", destinationPath)
	}

	sourcePath := "/usr/local/bin/configs"
	fileName := "commit-msg"
	src := fmt.Sprintf("%v/%v", sourcePath, fileName)
	dest := fmt.Sprintf("%v/%v", destinationPath, fileName)
	if err := util.CopyFile(src, dest); err != nil {
		return errors.Wrapf(err, "couldn't copy file %v", fileName)
	}
	return nil
}
