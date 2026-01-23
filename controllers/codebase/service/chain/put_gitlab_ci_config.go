package chain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitlabci "github.com/epam/edp-codebase-operator/v2/pkg/gitlab"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutGitLabCIConfig struct {
	client             client.Client
	gitlabCIManager    gitlabci.Manager
	gitProviderFactory gitproviderv2.GitProviderFactory
}

func NewPutGitLabCIConfig(
	c client.Client,
	m gitlabci.Manager,
	gitProviderFactory gitproviderv2.GitProviderFactory,
) *PutGitLabCIConfig {
	return &PutGitLabCIConfig{client: c, gitlabCIManager: m, gitProviderFactory: gitProviderFactory}
}

func (h *PutGitLabCIConfig) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	// Skip if not GitLab CI
	if codebase.Spec.CiTool != util.CIGitLab {
		log.Info("Skip GitLab CI config injection, not using GitLab CI")
		return nil
	}

	// Skip if already pushed (this stage or later)
	if codebase.Status.Git == util.ProjectGitLabCIPushedStatus ||
		codebase.Status.Git == util.ProjectTemplatesPushedStatus {
		log.Info("Skip GitLab CI config, already pushed in previous run")
		return nil
	}

	// Skip if already exists
	if h.gitlabCIAlreadyExists(codebase) {
		log.Info("Skip GitLab CI config, already exists in repository")
		return nil
	}

	log.Info("Start pushing GitLab CI config")

	if err := h.tryToPushGitLabCIConfig(ctx, codebase); err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to push GitLab CI config for %v codebase: %w", codebase.Name, err)
	}

	log.Info("End pushing GitLab CI config")

	// Set status to mark this stage complete
	if err := updateGitStatusWithPatch(
		ctx,
		h.client,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectGitLabCIPushedStatus,
	); err != nil {
		return err
	}

	log.Info("GitLab CI status has been set successfully")

	return nil
}

func (h *PutGitLabCIConfig) gitlabCIAlreadyExists(codebase *codebaseApi.Codebase) bool {
	wd := util.GetWorkDir(codebase.Name, codebase.Namespace)
	gitlabCIPath := filepath.Join(wd, gitlabci.GitLabCIFileName)

	// Check if file exists locally first
	if _, err := os.Stat(gitlabCIPath); err == nil {
		return true
	}

	return false
}

func (h *PutGitLabCIConfig) tryToPushGitLabCIConfig(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	// Prepare git repository (get server, clone, checkout)
	gitCtx, err := PrepareGitRepository(ctx, h.client, codebase, h.gitProviderFactory)
	if err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to prepare git repository: %w", err)
	}

	// Create git provider using factory
	g := h.gitProviderFactory(gitproviderv2.NewConfigFromGitServerAndSecret(gitCtx.GitServer, gitCtx.GitServerSecret))

	// Inject GitLab CI configuration
	log.Info("Start injecting GitLab CI config")

	err = h.gitlabCIManager.InjectGitLabCIConfig(ctx, codebase, gitCtx.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to inject GitLab CI config: %w", err)
	}

	log.Info("GitLab CI config has been injected")
	log.Info("Start committing changes")

	// Commit changes
	err = g.Commit(ctx, gitCtx.WorkDir, "Add GitLab CI configuration")
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

	log.Info("GitLab CI config has been pushed successfully")

	return nil
}
