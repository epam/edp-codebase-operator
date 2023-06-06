package put_branch_in_git

import (
	"context"
	"fmt"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutBranchInGit struct {
	Next    handler.CodebaseBranchHandler
	Client  client.Client
	Git     gitserver.Git
	Service service.CodebaseBranchService
}

var log = ctrl.Log.WithName("put-branch-in-git-chain")

func (h PutBranchInGit) ServeRequest(cb *codebaseApi.CodebaseBranch) error {
	rl := log.WithValues("namespace", cb.Namespace, "codebase branch", cb.Name)
	rl.Info("start PutBranchInGit method...")

	if err := h.setIntermediateSuccessFields(cb, codebaseApi.AcceptCodebaseBranchRegistration); err != nil {
		return err
	}

	c, err := util.GetCodebase(h.Client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		setFailedFields(cb, codebaseApi.PutGitBranch, err.Error())

		return fmt.Errorf("failed to fetch Codebase: %w", err)
	}

	if !c.Status.Available {
		log.Info("failed to start reconciling for branch; codebase is unavailable", "codebase", c.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase is unavailable", c.Name))
	}

	err = h.processNewVersion(cb, c)
	if err != nil {
		err = fmt.Errorf("failed to process new version for %s branch: %w", cb.Name, err)

		setFailedFields(cb, codebaseApi.PutGitBranch, err.Error())

		return err
	}

	gitServer := &codebaseApi.GitServer{}
	if err = h.Client.Get(
		context.TODO(),
		client.ObjectKey{
			Namespace: cb.Namespace,
			Name:      c.Spec.GitServer,
		},
		gitServer,
	); err != nil {
		setFailedFields(cb, codebaseApi.PutGitBranch, err.Error())

		return fmt.Errorf("failed to fetch GitServer: %w", err)
	}

	secret, err := util.GetSecret(h.Client, gitServer.Spec.NameSshKeySecret, c.Namespace)
	if err != nil {
		err = fmt.Errorf("failed to get %v secret: %w", gitServer.Spec.NameSshKeySecret, err)
		setFailedFields(cb, codebaseApi.PutGitBranch, err.Error())

		return err
	}

	wd := util.GetWorkDir(cb.Spec.CodebaseName, fmt.Sprintf("%v-%v", cb.Namespace, cb.Spec.BranchName))
	if !checkDirectory(wd) {
		repoSshUrl := util.GetSSHUrl(gitServer, c.Spec.GetProjectID())

		err = h.Git.CloneRepositoryBySsh(string(secret.Data[util.PrivateSShKeyName]), gitServer.Spec.GitUser, repoSshUrl, wd, gitServer.Spec.SshPort)
		if err != nil {
			setFailedFields(cb, codebaseApi.PutGitBranch, err.Error())

			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	err = h.Git.CreateRemoteBranch(string(secret.Data[util.PrivateSShKeyName]), gitServer.Spec.GitUser, wd, cb.Spec.BranchName, cb.Spec.FromCommit, gitServer.Spec.SshPort)
	if err != nil {
		setFailedFields(cb, codebaseApi.PutGitBranch, err.Error())

		return fmt.Errorf("failed to create remove branch: %w", err)
	}

	rl.Info("end PutBranchInGit method...")

	err = handler.NextServeOrNil(h.Next, cb)
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
	}

	err := h.Client.Status().Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch status field %q: %w", cb.Name, err)
	}

	err = h.Client.Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch resource %q: %w", cb.Name, err)
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
	}
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}

func (h PutBranchInGit) processNewVersion(codebaseBranch *codebaseApi.CodebaseBranch, codeBase *codebaseApi.Codebase) error {
	if codeBase.Spec.Versioning.Type != codebaseApi.VersioningTypeEDP {
		return nil
	}

	hasVersion, err := chain.HasNewVersion(codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to check if branch %v has new version: %w", codebaseBranch.Name, err)
	}

	if !hasVersion {
		return nil
	}

	err = h.Service.ResetBranchBuildCounter(codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to reset bulid counterL %w", err)
	}

	err = h.Service.ResetBranchSuccessBuildCounter(codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to reset success bulid counter: %w", err)
	}

	err = h.Service.AppendVersionToTheHistorySlice(codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to append version to hitory slice: %w", err)
	}

	return nil
}
