package chain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitlabci "github.com/epam/edp-codebase-operator/v2/pkg/gitlab"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutGitLabCIConfig struct {
	client          client.Client
	git             git.Git
	gitlabCIManager gitlabci.Manager
}

func NewPutGitLabCIConfig(c client.Client, g git.Git, m gitlabci.Manager) *PutGitLabCIConfig {
	return &PutGitLabCIConfig{client: c, git: g, gitlabCIManager: m}
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
	codebase.Status.Git = util.ProjectGitLabCIPushedStatus
	if err := h.client.Status().Update(ctx, codebase); err != nil {
		setFailedFields(codebase, codebaseApi.RepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to set git status %s for codebase %s: %w",
			util.ProjectGitLabCIPushedStatus, codebase.Name, err)
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

	gitServer := &codebaseApi.GitServer{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace}, gitServer); err != nil {
		return fmt.Errorf("failed to get GitServer: %w", err)
	}

	gitServerSecret := &corev1.Secret{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: gitServer.Spec.NameSshKeySecret, Namespace: codebase.Namespace}, gitServerSecret); err != nil {
		return fmt.Errorf("failed to get GitServer secret: %w", err)
	}

	privateSSHKey := string(gitServerSecret.Data[util.PrivateSShKeyName])
	repoSshUrl := util.GetSSHUrl(gitServer, codebase.Spec.GetProjectID())
	wd := util.GetWorkDir(codebase.Name, codebase.Namespace)

	// Ensure we have the repository locally
	if !util.DoesDirectoryExist(wd) || util.IsDirectoryEmpty(wd) {
		log.Info("Start cloning repository", "url", repoSshUrl)

		if err := h.git.CloneRepositoryBySsh(ctx, privateSSHKey, gitServer.Spec.GitUser, repoSshUrl, wd, gitServer.Spec.SshPort); err != nil {
			return fmt.Errorf("failed to clone git repository: %w", err)
		}

		log.Info("Repository has been cloned", "url", repoSshUrl)
	}

	ru, err := util.GetRepoUrl(codebase)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	log.Info("Start checkout default branch", "branch", codebase.Spec.DefaultBranch, "repo", ru)

	err = CheckoutBranch(ru, wd, codebase.Spec.DefaultBranch, h.git, codebase, h.client)
	if err != nil {
		return fmt.Errorf("failed to checkout default branch %v: %w", codebase.Spec.DefaultBranch, err)
	}

	log.Info("Default branch has been checked out")
	log.Info("Start injecting GitLab CI config")

	// Inject the GitLab CI configuration
	err = h.gitlabCIManager.InjectGitLabCIConfig(ctx, codebase, wd)
	if err != nil {
		return fmt.Errorf("failed to inject GitLab CI config: %w", err)
	}

	log.Info("GitLab CI config has been injected")
	log.Info("Start committing changes")

	err = h.git.CommitChanges(wd, "Add GitLab CI configuration")
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Info("Changes have been committed")
	log.Info("Start pushing changes")

	err = h.git.PushChanges(privateSSHKey, gitServer.Spec.GitUser, wd, gitServer.Spec.SshPort, "--all")
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info("GitLab CI config has been pushed successfully")

	return nil
}
