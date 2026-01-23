package chain

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// GitRepositoryContext holds all context needed for git operations.
// It contains the GitServer configuration, credentials, and paths
// required for SSH-based git operations.
type GitRepositoryContext struct {
	GitServer       *codebaseApi.GitServer
	GitServerSecret *corev1.Secret
	PrivateSSHKey   string
	UserName        string
	Token           string
	RepoGitUrl      string
	WorkDir         string
}

func setIntermediateSuccessFields(
	ctx context.Context,
	c client.Client,
	cb *codebaseApi.Codebase,
	action codebaseApi.ActionType,
) error {
	// Set WebHookRef from WebHookID for backward compatibility.
	webHookRef := cb.Status.WebHookRef
	if webHookRef == "" && cb.Status.WebHookID != 0 {
		webHookRef = strconv.Itoa(cb.Status.WebHookID)
	}

	cb.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Action:          action,
		Result:          codebaseApi.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    cb.Status.FailureCount,
		Git:             cb.Status.Git,
		WebHookID:       cb.Status.WebHookID,
		WebHookRef:      webHookRef,
		GitWebUrl:       cb.Status.GitWebUrl,
	}

	if err := c.Status().Update(ctx, cb); err != nil {
		return fmt.Errorf("failed to update status field of %q resource 'Codebase': %w", cb.Name, err)
	}

	return nil
}

// updateGitStatusWithPatch updates the codebase Git status using Patch instead of Update.
// If a conflict occurs, the function returns an error, causing the reconciliation
// to requeue automatically via the controller-runtime framework.
func updateGitStatusWithPatch(
	ctx context.Context,
	c client.Client,
	codebase *codebaseApi.Codebase,
	action codebaseApi.ActionType,
	gitStatus string,
) error {
	// Skip update if status already matches (idempotency check)
	if codebase.Status.Git == gitStatus {
		return nil
	}

	// Create patch based on current object state
	patch := client.MergeFrom(codebase.DeepCopy())

	// Modify the status field
	codebase.Status.Git = gitStatus

	// Apply patch to status subresource
	if err := c.Status().Patch(ctx, codebase, patch); err != nil {
		setFailedFields(codebase, action, err.Error())

		return fmt.Errorf("failed to patch git status to %s for codebase %s: %w",
			gitStatus, codebase.Name, err)
	}

	return nil
}

// PrepareGitRepository performs complete git repository preparation workflow:
// 1. Retrieves GitServer resource and its Secret
// 2. Extracts SSH credentials and builds repository paths
// 3. Clones repository via SSH if not already present locally
// 4. Checks out the codebase's default branch
//
// This function handles the common git setup operations shared across
// multiple chain handlers. Provider-specific operations (e.g., Gerrit hooks)
// should be handled by the caller after this function returns.
//
// Returns GitRepositoryContext containing all necessary context for
// subsequent git operations (commit, push, etc.).
func PrepareGitRepository(
	ctx context.Context,
	c client.Client,
	codebase *codebaseApi.Codebase,
	gitProviderFactory gitproviderv2.GitProviderFactory,
) (*GitRepositoryContext, error) {
	log := ctrl.LoggerFrom(ctx)

	gitRepoCtx, err := GetGitRepositoryContext(ctx, c, codebase)
	if err != nil {
		return nil, err
	}

	gitProvider := gitProviderFactory(
		gitproviderv2.NewConfigFromGitServerAndSecret(
			gitRepoCtx.GitServer,
			gitRepoCtx.GitServerSecret,
		),
	)

	if !util.DoesDirectoryExist(gitRepoCtx.WorkDir) || util.IsDirectoryEmpty(gitRepoCtx.WorkDir) {
		log.Info("Start cloning repository", "url", gitRepoCtx.RepoGitUrl)

		if err := gitProvider.Clone(ctx, gitRepoCtx.RepoGitUrl, gitRepoCtx.WorkDir); err != nil {
			return nil, fmt.Errorf("failed to clone git repository: %w", err)
		}

		log.Info("Repository has been cloned", "url", gitRepoCtx.RepoGitUrl)
	}

	log.Info("Start checkout default branch", "branch", codebase.Spec.DefaultBranch)

	err = CheckoutBranch(ctx, codebase.Spec.DefaultBranch, gitRepoCtx, codebase, c, gitProviderFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to checkout default branch %v: %w", codebase.Spec.DefaultBranch, err)
	}

	log.Info("Default branch has been checked out", "branch", codebase.Spec.DefaultBranch)

	return gitRepoCtx, nil
}

func GetGitRepositoryContext(
	ctx context.Context,
	c client.Client,
	codebase *codebaseApi.Codebase,
) (*GitRepositoryContext, error) {
	gitServer := &codebaseApi.GitServer{}
	if err := c.Get(
		ctx,
		client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace},
		gitServer,
	); err != nil {
		return nil, fmt.Errorf("failed to get GitServer: %w", err)
	}

	gitServerSecret := &corev1.Secret{}
	if err := c.Get(
		ctx,
		client.ObjectKey{Name: gitServer.Spec.NameSshKeySecret, Namespace: codebase.Namespace},
		gitServerSecret,
	); err != nil {
		return nil, fmt.Errorf("failed to get GitServer secret: %w", err)
	}

	return &GitRepositoryContext{
		GitServer:       gitServer,
		GitServerSecret: gitServerSecret,
		PrivateSSHKey:   string(gitServerSecret.Data[util.PrivateSShKeyName]),
		UserName:        string(gitServerSecret.Data[util.GitServerSecretUserNameField]),
		Token:           string(gitServerSecret.Data[util.GitServerSecretTokenField]),
		RepoGitUrl:      util.GetProjectGitUrl(gitServer, gitServerSecret, codebase.Spec.GetProjectID()),
		WorkDir:         util.GetWorkDir(codebase.Name, codebase.Namespace),
	}, nil
}
