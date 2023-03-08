package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/template"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutDeployConfigsToGitProvider struct {
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func NewPutDeployConfigsToGitProvider(c client.Client, cr repository.CodebaseRepository, g git.Git) *PutDeployConfigsToGitProvider {
	return &PutDeployConfigsToGitProvider{client: c, cr: cr, git: g}
}

func (h *PutDeployConfigsToGitProvider) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)

	if c.Spec.DisablePutDeployTemplates {
		rLog.Info("skip of putting deploy templates to codebase due to specified flag")
		return nil
	}

	rLog.Info("Start pushing configs...")

	if err := h.tryToPushConfigs(ctx, c); err != nil {
		setFailedFields(c, codebaseApi.SetupDeploymentTemplates, err.Error())
		return fmt.Errorf("failed to push deploy configs for %v codebase: %w", c.Name, err)
	}

	rLog.Info("end pushing configs to remote git server")

	return nil
}

func (h *PutDeployConfigsToGitProvider) tryToPushConfigs(ctx context.Context, c *codebaseApi.Codebase) error {
	name, err := helper.GetEDPName(ctx, h.client, c.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get edp name: %w", err)
	}

	skip, err := h.skipTemplatePreparing(ctx, *name, c.Name)
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
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	err = CheckoutBranch(ru, wd, c.Spec.DefaultBranch, h.git, c, h.client)
	if err != nil {
		return fmt.Errorf("checkout default branch %v in Git put_deploy_config has been failed: %w", c.Spec.DefaultBranch, err)
	}

	err = template.PrepareTemplates(h.client, c, wd, ad)
	if err != nil {
		return fmt.Errorf("failed to prepare templates: %w", err)
	}

	err = h.git.CommitChanges(wd, fmt.Sprintf("Add deployment templates for %v", c.Name))
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := pushChangesToGit(h.client, h.git, wd, c); err != nil {
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(ctx, util.ProjectTemplatesPushedStatus, c.Name, *name); err != nil {
		return fmt.Errorf("failed to set project_status %v value for %v codebase: %w", util.ProjectTemplatesPushedStatus, c.Name, err)
	}

	return nil
}

func (h *PutDeployConfigsToGitProvider) skipTemplatePreparing(ctx context.Context, edpName, codebaseName string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(ctx, codebaseName, edpName)
	if err != nil {
		return true, fmt.Errorf("failed to get project_status value for %v codebase: %w", codebaseName, err)
	}

	if util.ContainsString([]string{util.ProjectTemplatesPushedStatus}, ps) {
		return true, nil
	}

	return false, nil
}
