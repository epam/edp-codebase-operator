package put_branch_in_git

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
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
		setFailedFields(cb, codebaseApi.PutBranchForGitlabCiCodebase, err.Error())

		return fmt.Errorf("failed to fetch Codebase: %w", err)
	}

	if !c.Status.Available {
		log.Info("couldn't start reconciling for branch. codebase is unavailable", "codebase", c.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase is unavailable", c.Name))
	}

	if c.Spec.Versioning.Type == util.VersioningTypeEDP && hasNewVersion(cb) {
		err = h.processNewVersion(cb)
		if err != nil {
			err = errors.Wrapf(err, "couldn't process new version for %v branch", cb.Name)

			setFailedFields(cb, codebaseApi.PutBranchForGitlabCiCodebase, err.Error())

			return err
		}
	}

	gs, err := util.GetGitServer(h.Client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(cb, codebaseApi.PutBranchForGitlabCiCodebase, err.Error())

		return fmt.Errorf("failed to fetch GitServer: %w", err)
	}

	secret, err := util.GetSecret(h.Client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		err = errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
		setFailedFields(cb, codebaseApi.PutBranchForGitlabCiCodebase, err.Error())

		return err
	}

	wd := util.GetWorkDir(cb.Spec.CodebaseName, fmt.Sprintf("%v-%v", cb.Namespace, cb.Spec.BranchName))
	if !checkDirectory(wd) {
		// we work with gerrit by default
		ru := fmt.Sprintf("%v:%v", gs.GitHost, c.Name)
		// may be it's third party VCS
		if c.Spec.GitUrlPath != nil {
			ru = fmt.Sprintf("%v:%v", gs.GitHost, *c.Spec.GitUrlPath)
		}

		err = h.Git.CloneRepositoryBySsh(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, ru, wd, gs.SshPort)
		if err != nil {
			setFailedFields(cb, codebaseApi.PutBranchForGitlabCiCodebase, err.Error())

			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	err = h.Git.CreateRemoteBranch(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, wd, cb.Spec.BranchName, cb.Spec.FromCommit, gs.SshPort)
	if err != nil {
		setFailedFields(cb, codebaseApi.PutBranchForGitlabCiCodebase, err.Error())

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
	}
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}

func (h PutBranchInGit) processNewVersion(b *codebaseApi.CodebaseBranch) error {
	err := h.Service.ResetBranchBuildCounter(b)
	if err != nil {
		return fmt.Errorf("failed to reset bulid counterL %w", err)
	}

	err = h.Service.ResetBranchSuccessBuildCounter(b)
	if err != nil {
		return fmt.Errorf("failed to reset success bulid counter: %w", err)
	}

	err = h.Service.AppendVersionToTheHistorySlice(b)
	if err != nil {
		return fmt.Errorf("failed to append version to hitory slice: %w", err)
	}

	return nil
}

func hasNewVersion(b *codebaseApi.CodebaseBranch) bool {
	return !util.SearchVersion(b.Status.VersionHistory, *b.Spec.Version)
}
