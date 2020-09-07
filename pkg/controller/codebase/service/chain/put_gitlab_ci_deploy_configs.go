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

type PutGitlabCiDeployConfigs struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
	cr        repository.CodebaseRepository
	git       git.Git
}

func (h PutGitlabCiDeployConfigs) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start pushing configs...")

	if err := h.tryToPushConfigs(*c); err != nil {
		setFailedFields(c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs for %v codebase", c.Name)
	}
	rLog.Info("end pushing configs to remote git server")
	return nextServeOrNil(h.next, c)
}

func (h PutGitlabCiDeployConfigs) tryToPushConfigs(c v1alpha1.Codebase) error {
	name, err := helper.GetEDPName(h.clientSet.Client, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get edp name")
	}

	skip, err := h.skipTemplatePreparing(*name, c.Name, c.Namespace)
	if err != nil {
		return err
	}

	if skip {
		log.V(2).Info("skip pushing templates to git project. templates already pushed",
			"name", c.Name)
		return nil
	}

	if err := template.PrepareGitlabCITemplates(h.clientSet.CoreClient, c); err != nil {
		return err
	}

	gf := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v/%v", c.Namespace, c.Name, "templates", c.Name)
	if err := h.git.CommitChanges(gf, fmt.Sprintf("Add template for %v", c.Name)); err != nil {
		return err
	}

	if err := h.pushChanges(gf, c.Spec.GitServer, c.Namespace); err != nil {
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectTemplatesPushedStatus, c.Name, *name); err != nil {
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	return nil
}

func (h PutGitlabCiDeployConfigs) pushChanges(projectPath, gitServerName, namespace string) error {
	gs, err := util.GetGitServer(h.clientSet.Client, gitServerName, namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(*h.clientSet.CoreClient, gs.NameSshKeySecret, namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := h.git.PushChanges(k, u, projectPath); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v repo", projectPath)
	}
	log.Info("templates have been pushed")
	return nil
}

func (h PutGitlabCiDeployConfigs) skipTemplatePreparing(edpName, codebaseName, namespace string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(codebaseName, edpName)
	if err != nil {
		return true, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if util.ContainsString([]string{util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus, util.GitlabCiFilePushedStatus}, *ps) {
		return true, nil
	}
	return false, nil
}
