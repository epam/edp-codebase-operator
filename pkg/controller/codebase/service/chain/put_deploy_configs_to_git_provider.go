package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/template"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
)

type PutDeployConfigsToGitProvider struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
	cr        repository.CodebaseRepository
	git       git.Git
}

func (h PutDeployConfigsToGitProvider) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start pushing configs...")

	if err := h.tryToPushConfigs(*c); err != nil {
		setFailedFields(c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs for %v codebase", c.Name)
	}
	rLog.Info("end pushing configs to remote git server")
	return nextServeOrNil(h.next, c)
}

func (h PutDeployConfigsToGitProvider) tryToPushConfigs(c v1alpha1.Codebase) error {
	edpN, err := helper.GetEDPName(h.clientSet.Client, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(c.Name, *edpN)
	if err != nil {
		return errors.Wrapf(err, "couldn't get project_status value for %v codebase", c.Name)
	}

	var status = []string{util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}
	if util.ContainsString(status, *ps) {
		log.V(2).Info("skip pushing templates to gerrit. teplates already pushed", "name", c.Name)
		return nil
	}

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(*h.clientSet.CoreClient, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	td := fmt.Sprintf("%v/%v", wd, "templates")
	gf := fmt.Sprintf("%v/%v", td, c.Name)
	log.Info("path to local Git folder", "path", gf)

	if err := template.PrepareTemplates(h.clientSet.CoreClient, c); err != nil {
		return err
	}

	if err := h.git.CommitChanges(gf, fmt.Sprintf("Add template for %v", c.Name)); err != nil {
		return err
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := h.git.PushChanges(k, u, gf); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v codebase", c.Name)
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectTemplatesPushedStatus, c.Name, *edpN); err != nil {
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	return nil
}
