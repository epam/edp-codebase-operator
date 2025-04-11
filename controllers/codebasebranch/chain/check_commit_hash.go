package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// CheckCommitHashExists is chain element for checking commit hash existence.
type CheckCommitHashExists struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
	Git    git.Git
}

// ServeRequest is a method for checking CodebaseBranch FromCommit hash existence.
func (c CheckCommitHashExists) ServeRequest(ctx context.Context, codebaseBranch *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx).WithName("check-commit-hash-exists")

	if codebaseBranch.Status.Git == codebaseApi.CodebaseBranchGitStatusBranchCreated {
		log.Info("Branch is already created in git. Skip checking commit hash existence")

		return c.next(ctx, codebaseBranch)
	}

	if codebaseBranch.Spec.FromCommit == "" {
		return c.next(ctx, codebaseBranch)
	}

	codebase := &codebaseApi.Codebase{}
	if err := c.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: codebaseBranch.Namespace,
			Name:      codebaseBranch.Spec.CodebaseName,
		},
		codebase,
	); err != nil {
		return c.processErr(codebaseBranch, fmt.Errorf("failed to get codebase %s: %w", codebaseBranch.Spec.CodebaseName, err))
	}

	gitServer := &codebaseApi.GitServer{}
	if err := c.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: codebaseBranch.Namespace,
			Name:      codebase.Spec.GitServer,
		},
		gitServer,
	); err != nil {
		return c.processErr(codebaseBranch, fmt.Errorf("failed to get git server %s: %w", codebase.Spec.GitServer, err))
	}

	secret := &corev1.Secret{}
	if err := c.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: codebaseBranch.Namespace,
			Name:      gitServer.Spec.NameSshKeySecret,
		},
		secret,
	); err != nil {
		return c.processErr(codebaseBranch, fmt.Errorf("failed to get secret %s: %w", gitServer.Spec.NameSshKeySecret, err))
	}

	workDir := GetCodebaseBranchWorkingDirectory(codebaseBranch)
	if !DirectoryExistsNotEmpty(workDir) {
		repoSshUrl := util.GetSSHUrl(gitServer, codebase.Spec.GetProjectID())

		if err := c.Git.CloneRepositoryBySsh(
			ctx,
			string(secret.Data[util.PrivateSShKeyName]),
			gitServer.Spec.GitUser,
			repoSshUrl,
			workDir,
			gitServer.Spec.SshPort,
		); err != nil {
			return c.processErr(codebaseBranch, fmt.Errorf("failed to clone repository: %w", err))
		}
	}

	exists, err := c.Git.CommitExists(workDir, codebaseBranch.Spec.FromCommit)
	if err != nil {
		return c.processErr(codebaseBranch, fmt.Errorf("failed to check commit hash %s: %w", codebaseBranch.Spec.FromCommit, err))
	}

	if !exists {
		return c.processErr(codebaseBranch, fmt.Errorf("commit %s doesn't exist", codebaseBranch.Spec.FromCommit))
	}

	return c.next(ctx, codebaseBranch)
}

// next is a method for serving next chain element.
func (c CheckCommitHashExists) next(ctx context.Context, codebaseBranch *codebaseApi.CodebaseBranch) error {
	err := handler.NextServeOrNil(ctx, c.Next, codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to serve next chain element: %w", err)
	}

	return nil
}

// processErr is a method for processing error in chain.
func (c CheckCommitHashExists) processErr(codebaseBranch *codebaseApi.CodebaseBranch, err error) error {
	if err == nil {
		return nil
	}

	c.setFailedFields(codebaseBranch, codebaseApi.CheckCommitHashExists, err.Error())

	return err
}

func (CheckCommitHashExists) setFailedFields(
	codebaseBranch *codebaseApi.CodebaseBranch,
	actionType codebaseApi.ActionType,
	message string,
) {
	codebaseBranch.Status = codebaseApi.CodebaseBranchStatus{
		Status:              util.StatusFailed,
		LastTimeUpdated:     metav1.Now(),
		Username:            "system",
		Action:              actionType,
		Result:              codebaseApi.Error,
		DetailedMessage:     message,
		Value:               "failed",
		VersionHistory:      codebaseBranch.Status.VersionHistory,
		LastSuccessfulBuild: codebaseBranch.Status.LastSuccessfulBuild,
		Build:               codebaseBranch.Status.Build,
		FailureCount:        codebaseBranch.Status.FailureCount,
		Git:                 codebaseBranch.Status.Git,
	}
}
