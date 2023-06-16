package trigger_job

import (
	"context"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("trigger-job-chain")

type TriggerJob struct {
	Client  client.Client
	Service service.CodebaseBranchService
	Next    handler.CodebaseBranchHandler
}

func (h TriggerJob) Trigger(ctx context.Context, cb *codebaseApi.CodebaseBranch, actionType codebaseApi.ActionType,
	triggerFunc func(cb *codebaseApi.CodebaseBranch) error,
) error {
	if err := h.SetIntermediateSuccessFields(cb, actionType); err != nil {
		return err
	}

	c, err := util.GetCodebase(h.Client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		h.SetFailedFields(cb, actionType, err.Error())

		return fmt.Errorf("failed to fetch codebase resource: %w", err)
	}

	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")

	jf, err := h.GetJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		h.SetFailedFields(cb, actionType, err.Error())
		return err
	}

	if !c.Status.Available || !isJenkinsFolderAvailable(jf) {
		log.Info("failed to start reconciling for branch. someone of codebase or jenkins folder is unavailable",
			"codebase", c.Name, "branch", cb.Name)

		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase and/or jenkinsfolder %v are/is unavailable", c.Name, jfn))
	}

	err = h.ProcessNewVersion(cb, c)
	if err != nil {
		h.SetFailedFields(cb, actionType, err.Error())

		return fmt.Errorf("failed to process new version for %v branch: %w", cb.Name, err)
	}

	err = triggerFunc(cb)
	if err != nil {
		h.SetFailedFields(cb, actionType, err.Error())
		return err
	}

	err = handler.NextServeOrNil(ctx, h.Next, cb)
	if err != nil {
		return fmt.Errorf("failed to process next handler in chain: %w", err)
	}

	return nil
}

func (h TriggerJob) SetIntermediateSuccessFields(cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
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
		return fmt.Errorf("failed to update CodebaseBranchStatus status field %q: %w", cb.Name, err)
	}

	err = h.Client.Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("SetIntermediateSuccessFields failed for %v branch: %w", cb.Name, err)
	}

	return nil
}

func (TriggerJob) SetFailedFields(cb *codebaseApi.CodebaseBranch, a codebaseApi.ActionType, message string) {
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

func (h TriggerJob) GetJenkinsFolder(name, namespace string) (*jenkinsApi.JenkinsFolder, error) {
	i := &jenkinsApi.JenkinsFolder{}

	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, i)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get jenkins folder %v: %w", name, err)
	}

	return i, nil
}

func (h TriggerJob) ProcessNewVersion(codebaseBranch *codebaseApi.CodebaseBranch, codeBase *codebaseApi.Codebase) error {
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

	if err = h.Service.ResetBranchBuildCounter(codebaseBranch); err != nil {
		return fmt.Errorf("failed reset branch build counter: %w", err)
	}

	if err = h.Service.ResetBranchSuccessBuildCounter(codebaseBranch); err != nil {
		return fmt.Errorf("failed reset branch success build counter: %w", err)
	}

	err = h.Service.AppendVersionToTheHistorySlice(codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to append version to resource: %w", err)
	}

	return nil
}

func isJenkinsFolderAvailable(jf *jenkinsApi.JenkinsFolder) bool {
	if jf == nil {
		return false
	}

	return jf.Status.Available
}
