package chain

import (
	"fmt"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/template"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutDeployConfigsToGitProvider struct {
	next   handler.CodebaseHandler
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func (h PutDeployConfigsToGitProvider) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	if c.Spec.DisablePutDeployTemplates {
		rLog.Info("skip of putting deploy templates to codebase due to specified flag")
		return nextServeOrNil(h.next, c)
	}

	rLog.Info("Start pushing configs...")
	if err := h.tryToPushConfigs(c); err != nil {
		setFailedFields(c, v1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs for %v codebase", c.Name)
	}
	rLog.Info("end pushing configs to remote git server")
	return nextServeOrNil(h.next, c)
}

func (h PutDeployConfigsToGitProvider) tryToPushConfigs(c *v1alpha1.Codebase) error {
	name, err := helper.GetEDPName(h.client, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get edp name")
	}

	skip, err := h.skipTemplatePreparing(*name, c.Name, c.Namespace)
	if err != nil {
		return err
	}

	if skip {
		log.Info("skip pushing templates. templates already pushed", "name", c.Name)
		return nil
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	ad := util.GetAssetsDir()
	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	if err := CheckoutBranch(ru, wd, c.Spec.DefaultBranch, h.git, c, h.client); err != nil {
		return errors.Wrapf(err, "checkout default branch %v in Git put_deploy_config has been failed", c.Spec.DefaultBranch)
	}

	if err := template.PrepareTemplates(h.client, c, wd, ad); err != nil {
		return err
	}

	if err := h.git.CommitChanges(wd, fmt.Sprintf("Add deployment templates for %v", c.Name)); err != nil {
		return err
	}

	if err := h.pushChanges(wd, c.Spec.GitServer, c.Namespace, c.Spec.DefaultBranch); err != nil {
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectTemplatesPushedStatus, c.Name, *name); err != nil {
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	return nil
}

func (h PutDeployConfigsToGitProvider) pushChanges(projectPath, gitServerName, namespace, defaultBranch string) error {
	gs, err := util.GetGitServer(h.client, gitServerName, namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := h.git.PushChanges(k, u, projectPath, defaultBranch); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v repo", projectPath)
	}
	log.Info("templates have been pushed")
	return nil
}

func (h PutDeployConfigsToGitProvider) skipTemplatePreparing(edpName, codebaseName, namespace string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(codebaseName, edpName)
	if err != nil {
		return true, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if util.ContainsString([]string{util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}, *ps) {
		return true, nil
	}
	return false, nil
}
