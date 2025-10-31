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
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// CheckReferenceExists is chain element for checking if a reference (branch or commit) exists.
type CheckReferenceExists struct {
	Next               handler.CodebaseBranchHandler
	Client             client.Client
	GitProviderFactory gitproviderv2.GitProviderFactory
}

// ServeRequest is a method for checking if the reference (branch or commit) exists.
func (c CheckReferenceExists) ServeRequest(ctx context.Context, codebaseBranch *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx).WithName("check-reference-exists")

	if codebaseBranch.Status.Git == codebaseApi.CodebaseBranchGitStatusBranchCreated {
		log.Info("Branch is already created in git. Skip checking reference existence")

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

	// Create git provider using factory
	g := c.GitProviderFactory(gitServer, secret)

	workDir := GetCodebaseBranchWorkingDirectory(codebaseBranch)
	if !DirectoryExistsNotEmpty(workDir) {
		repoGitUrl := util.GetProjectGitUrl(gitServer, secret, codebase.Spec.GetProjectID())

		if err := g.Clone(ctx, repoGitUrl, workDir, 0); err != nil {
			return c.processErr(codebaseBranch, fmt.Errorf("failed to clone repository: %w", err))
		}
	}

	err := g.CheckReference(ctx, workDir, codebaseBranch.Spec.FromCommit)
	if err != nil {
		return c.processErr(codebaseBranch, fmt.Errorf("reference %s doesn't exist: %w", codebaseBranch.Spec.FromCommit, err))
	}

	return c.next(ctx, codebaseBranch)
}

// next is a method for serving next chain element.
func (c CheckReferenceExists) next(ctx context.Context, codebaseBranch *codebaseApi.CodebaseBranch) error {
	err := handler.NextServeOrNil(ctx, c.Next, codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to serve next chain element: %w", err)
	}

	return nil
}

// processErr is a method for processing error in chain.
func (c CheckReferenceExists) processErr(codebaseBranch *codebaseApi.CodebaseBranch, err error) error {
	if err == nil {
		return nil
	}

	c.setFailedFields(codebaseBranch, codebaseApi.CheckCommitHashExists, err.Error())

	return err
}

func (CheckReferenceExists) setFailedFields(
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
