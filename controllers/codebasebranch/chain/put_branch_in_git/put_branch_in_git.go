package put_branch_in_git

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutBranchInGit struct {
	Next    handler.CodebaseBranchHandler
	Client  client.Client
	Git     git.Git
	Service service.CodebaseBranchService
}

func (h PutBranchInGit) ServeRequest(ctx context.Context, branch *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx)

	if branch.Status.Git == codebaseApi.CodebaseBranchGitStatusBranchCreated {
		log.Info("Branch is already created in git")

		if err := handler.NextServeOrNil(ctx, h.Next, branch); err != nil {
			return fmt.Errorf("failed to process next handler in chain: %w", err)
		}

		return nil
	}

	log.Info("Start creating branch in git")

	if err := h.setIntermediateSuccessFields(branch, codebaseApi.AcceptCodebaseBranchRegistration); err != nil {
		return err
	}

	codebase := &codebaseApi.Codebase{}
	if err := h.Client.Get(ctx, client.ObjectKey{
		Namespace: branch.Namespace,
		Name:      branch.Spec.CodebaseName,
	}, codebase); err != nil {
		setFailedFields(branch, codebaseApi.PutGitBranch, err.Error())

		return fmt.Errorf("failed to fetch Codebase: %w", err)
	}

	if !codebase.Status.Available {
		log.Info("failed to start reconciling for branch; codebase is unavailable", "codebase", codebase.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase is unavailable", codebase.Name))
	}

	gitServer := &codebaseApi.GitServer{}
	if err := h.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: branch.Namespace,
			Name:      codebase.Spec.GitServer,
		},
		gitServer,
	); err != nil {
		setFailedFields(branch, codebaseApi.PutGitBranch, err.Error())

		return fmt.Errorf("failed to fetch GitServer: %w", err)
	}

	secret := &corev1.Secret{}
	if err := h.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: branch.Namespace,
			Name:      gitServer.Spec.NameSshKeySecret,
		},
		secret,
	); err != nil {
		err = fmt.Errorf("failed to get %v secret: %w", gitServer.Spec.NameSshKeySecret, err)
		setFailedFields(branch, codebaseApi.PutGitBranch, err.Error())

		return err
	}

	wd := chain.GetCodebaseBranchWorkingDirectory(branch)
	if !checkDirectory(wd) {
		repoSshUrl := util.GetSSHUrl(gitServer, codebase.Spec.GetProjectID())

		if err := h.Git.CloneRepositoryBySsh(
			ctx,
			string(secret.Data[util.PrivateSShKeyName]),
			gitServer.Spec.GitUser,
			repoSshUrl,
			wd,
			gitServer.Spec.SshPort,
		); err != nil {
			setFailedFields(branch, codebaseApi.PutGitBranch, err.Error())

			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	currentBranchName, err := h.Git.GetCurrentBranchName(wd)
	if err != nil {
		return fmt.Errorf("failed to get current branch name: %w", err)
	}

	if currentBranchName != codebase.Spec.DefaultBranch {
		if err = h.Git.CheckoutRemoteBranchBySSH(string(secret.Data[util.PrivateSShKeyName]), gitServer.Spec.GitUser, wd, codebase.Spec.DefaultBranch); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s: %w", codebase.Spec.DefaultBranch, err)
		}
	}

	err = h.Git.CreateRemoteBranch(string(secret.Data[util.PrivateSShKeyName]), gitServer.Spec.GitUser, wd, branch.Spec.BranchName, branch.Spec.FromCommit, gitServer.Spec.SshPort)
	if err != nil {
		setFailedFields(branch, codebaseApi.PutGitBranch, err.Error())

		// We need to remove work directory if branch creation failed(push error).
		// Otherwise, the next time the branch creation will be skipped because local branch already exists.
		if err = util.RemoveDirectory(wd); err != nil {
			log.Error(err, "failed to remove directory", "path", wd)
		}

		return fmt.Errorf("failed to create remote branch: %w", err)
	}

	branch.Status.Git = codebaseApi.CodebaseBranchGitStatusBranchCreated
	if err = h.Client.Status().Update(ctx, branch); err != nil {
		branch.Status.Git = ""
		setFailedFields(branch, codebaseApi.PutGitBranch, err.Error())

		return fmt.Errorf("failed to update CodebaseBranch status: %w", err)
	}

	log.Info("Branch has been created in git")

	err = handler.NextServeOrNil(ctx, h.Next, branch)
	if err != nil {
		return fmt.Errorf("failed to process next handler in chain: %w", err)
	}

	return nil
}

func (h PutBranchInGit) setIntermediateSuccessFields(cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
	ctx := context.Background()
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:              model.StatusInit,
		LastTimeUpdated:     metaV1.Now(),
		Action:              action,
		Result:              codebaseApi.Success,
		Username:            "system",
		Value:               "inactive",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
		FailureCount:        cb.Status.FailureCount,
		Git:                 cb.Status.Git,
	}

	err := h.Client.Status().Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch status field %q: %w", cb.Name, err)
	}

	return nil
}

func setFailedFields(cb *codebaseApi.CodebaseBranch, a codebaseApi.ActionType, message string) {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:              util.StatusFailed,
		LastTimeUpdated:     metaV1.Now(),
		Username:            "system",
		Action:              a,
		Result:              codebaseApi.Error,
		DetailedMessage:     message,
		Value:               "failed",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
		FailureCount:        cb.Status.FailureCount,
		Git:                 cb.Status.Git,
	}
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}
