package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type PutBranchInGit struct {
	next    handler.CodebaseBranchHandler
	client  client.Client
	git     gitserver.Git
	service service.CodebaseBranchService
}

func (h PutBranchInGit) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	rl := log.WithValues("namespace", cb.Namespace, "codebase branch", cb.Name)
	rl.Info("start PutBranchInGit method...")

	c, err := util.GetCodebase(h.client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		setFailedFields(cb, v1alpha1.PutBranchForGitlabCiCodebase, err.Error())
		return err
	}

	if !c.Status.Available {
		log.Info("couldn't start reconciling for branch. codebase is unavailable", "codebase", c.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase is unavailable", c.Name))
	}

	if c.Spec.Versioning.Type == util.VersioningTypeEDP && hasNewVersion(cb) {
		if err := h.processNewVersion(cb); err != nil {
			err = errors.Wrapf(err, "couldn't process new version for %v branch", cb.Name)
			setFailedFields(cb, v1alpha1.PutBranchForGitlabCiCodebase, err.Error())
			return err
		}
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(cb, v1alpha1.PutBranchForGitlabCiCodebase, err.Error())
		return err
	}

	secret, err := util.GetSecretData(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		err = errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
		setFailedFields(cb, v1alpha1.PutBranchForGitlabCiCodebase, err.Error())
		return err
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v", cb.Namespace, cb.Spec.CodebaseName, cb.Spec.BranchName)
	if !checkDirectory(wd) {
		ru := fmt.Sprintf("%v:%v%v", gs.GitHost, gs.SshPort, *c.Spec.GitUrlPath)
		if err := h.git.CloneRepositoryBySsh(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, ru, wd); err != nil {
			setFailedFields(cb, v1alpha1.PutBranchForGitlabCiCodebase, err.Error())
			return err
		}
	}

	if err := h.git.CreateRemoteBranch(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, wd, cb.Spec.BranchName); err != nil {
		setFailedFields(cb, v1alpha1.PutBranchForGitlabCiCodebase, err.Error())
		return err
	}
	rl.Info("end PutBranchInGit method...")
	return nextServeOrNil(h.next, cb)
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}

func (h PutBranchInGit) processNewVersion(b *v1alpha1.CodebaseBranch) error {
	if err := h.service.ResetBranchBuildCounter(b); err != nil {
		return err
	}

	if err := h.service.ResetBranchSuccessBuildCounter(b); err != nil {
		return err
	}

	return h.service.AppendVersionToTheHistorySlice(b)
}

func setFailedFields(cb *v1alpha1.CodebaseBranch, a v1alpha1.ActionType, message string) {
	cb.Status = v1alpha1.CodebaseBranchStatus{
		Status:          util.StatusFailed,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          a,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}
}
