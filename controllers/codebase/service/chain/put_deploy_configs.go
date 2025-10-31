package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/template"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutDeployConfigs struct {
	client                      client.Client
	gitProviderFactory          gitproviderv2.GitProviderFactory
	createGitProviderWithConfig func(config gitproviderv2.Config) gitproviderv2.Git
}

func NewPutDeployConfigs(c client.Client, gitProviderFactory gitproviderv2.GitProviderFactory, createGitProviderWithConfig func(config gitproviderv2.Config) gitproviderv2.Git) *PutDeployConfigs {
	return &PutDeployConfigs{client: c, gitProviderFactory: gitProviderFactory, createGitProviderWithConfig: createGitProviderWithConfig}
}

func (h *PutDeployConfigs) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if c.Spec.DisablePutDeployTemplates {
		log.Info("Skip of putting deploy templates to codebase due to specified flag")
		return nil
	}

	if c.Spec.Type != util.Application {
		log.Info("Skip putting deploy templates to codebase because it is not application")
		return nil
	}

	log.Info("Start pushing configs")

	if err := h.tryToPushConfigs(ctx, c); err != nil {
		setFailedFields(c, codebaseApi.SetupDeploymentTemplates, err.Error())
		return fmt.Errorf("failed to push deploy configs for %v codebase: %w", c.Name, err)
	}

	log.Info("End pushing configs")

	return nil
}

func (h *PutDeployConfigs) tryToPushConfigs(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if codebase.Status.Git == util.ProjectTemplatesPushedStatus {
		log.Info("Skip pushing templates. Templates already pushed")

		return nil
	}

	// Prepare git repository (get server, clone, checkout)
	gitCtx, err := PrepareGitRepository(ctx, h.client, codebase, h.gitProviderFactory, h.createGitProviderWithConfig)
	if err != nil {
		setFailedFields(codebase, codebaseApi.SetupDeploymentTemplates, err.Error())
		return fmt.Errorf("failed to prepare git repository: %w", err)
	}

	// Create git provider using factory
	g := h.gitProviderFactory(gitCtx.GitServer, gitCtx.GitServerSecret)

	// Add Gerrit-specific commit hooks if needed
	if gitCtx.GitServer.Spec.GitProvider == codebaseApi.GitProviderGerrit {
		log.Info("Start adding commit hooks")

		if err := h.addCommitHooks(gitCtx.WorkDir); err != nil {
			return fmt.Errorf("failed to add commit hooks: %w", err)
		}

		log.Info("Commit hooks have been added")
	}

	// Prepare deployment templates
	log.Info("Start preparing templates")

	err = template.PrepareTemplates(ctx, h.client, codebase, gitCtx.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to prepare template: %w", err)
	}

	log.Info("Templates have been prepared")
	log.Info("Start committing changes")

	// Commit changes
	err = g.Commit(ctx, gitCtx.WorkDir, fmt.Sprintf("Add deployment templates for %s", codebase.Name))
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Info("Changes have been committed")
	log.Info("Start pushing changes")

	// Push changes
	err = g.Push(ctx, gitCtx.WorkDir, gitproviderv2.RefSpecPushAllBranches)
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info("Changes have been pushed")

	// Update status
	if err = updateGitStatusWithPatch(ctx, h.client, codebase, codebaseApi.SetupDeploymentTemplates, util.ProjectTemplatesPushedStatus); err != nil {
		return err
	}

	log.Info("Config has been pushed")

	return nil
}

func (*PutDeployConfigs) addCommitHooks(wd string) error {
	destinationPath := fmt.Sprintf("%v/.git/hooks", wd)

	if err := util.CreateDirectory(destinationPath); err != nil {
		return fmt.Errorf("failed to create folder %v: %w", destinationPath, err)
	}

	assetsDir, err := util.GetAssetsDir()
	if err != nil {
		return fmt.Errorf("failed to get assets dir: %w", err)
	}

	fileName := "commit-msg"
	src := fmt.Sprintf("%v/configs/%v", assetsDir, fileName)
	dest := fmt.Sprintf("%v/%v", destinationPath, fileName)

	if err := util.CopyFile(src, dest); err != nil {
		return fmt.Errorf("failed to copy file %v: %w", fileName, err)
	}

	return nil
}
