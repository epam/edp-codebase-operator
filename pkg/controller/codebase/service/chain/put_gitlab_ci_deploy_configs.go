package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/template"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutGitlabCiDeployConfigs struct {
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func NewPutGitlabCiDeployConfigs(client client.Client, cr repository.CodebaseRepository, git git.Git) *PutGitlabCiDeployConfigs {
	return &PutGitlabCiDeployConfigs{client: client, cr: cr, git: git}
}

func (h *PutGitlabCiDeployConfigs) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	if c.Spec.DisablePutDeployTemplates {
		rLog.Info("skip of putting deploy templates to codebase due to specified flag")
		return nil
	}

	rLog.Info("Start pushing configs...")
	if err := h.tryToPushConfigs(ctx, c); err != nil {
		setFailedFields(c, codebaseApi.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs for %v codebase", c.Name)
	}
	rLog.Info("end pushing configs to remote git server")
	return nil
}

func (h *PutGitlabCiDeployConfigs) tryToPushConfigs(ctx context.Context, c *codebaseApi.Codebase) error {
	name, err := helper.GetEDPName(ctx, h.client, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get edp name")
	}

	skip, err := h.skipTemplatePreparing(*name, c.Name, c.Namespace)
	if err != nil {
		return err
	}

	if skip {
		log.Info("skip pushing templates to git project. templates already pushed",
			"name", c.Name)
		return nil
	}
	wd := util.GetWorkDir(c.Name, c.Namespace)
	ad := util.GetAssetsDir()

	if err := template.PrepareGitlabCITemplates(h.client, c, wd, ad); err != nil {
		return err
	}

	if err := h.git.CommitChanges(wd, fmt.Sprintf("Add template for %v", c.Name)); err != nil {
		return err
	}

	if err := pushChangesToGit(h.client, h.git, wd, c); err != nil {
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectTemplatesPushedStatus, c.Name, *name); err != nil {
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	return nil
}

func (h *PutGitlabCiDeployConfigs) skipTemplatePreparing(edpName, codebaseName, namespace string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(codebaseName, edpName)
	if err != nil {
		return true, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if util.ContainsString([]string{util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus, util.GitlabCiFilePushedStatus}, ps) {
		return true, nil
	}
	return false, nil
}
