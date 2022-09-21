package trigger_job

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

var log = ctrl.Log.WithName("trigger-job-chain")

type TriggerJob struct {
	Client  client.Client
	Service service.CodebaseBranchService
	Next    handler.CodebaseBranchHandler
}

func (h TriggerJob) Trigger(cb *codebaseApi.CodebaseBranch, actionType codebaseApi.ActionType,
	triggerFunc func(cb *codebaseApi.CodebaseBranch) error) error {

	if err := h.SetIntermediateSuccessFields(cb, actionType); err != nil {
		return err
	}

	c, err := util.GetCodebase(h.Client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		h.SetFailedFields(cb, actionType, err.Error())
		return err
	}

	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jf, err := h.GetJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		h.SetFailedFields(cb, actionType, err.Error())
		return err
	}

	if !c.Status.Available || !isJenkinsFolderAvailable(jf) {
		log.Info("couldn't start reconciling for branch. someone of codebase or jenkins folder is unavailable",
			"codebase", c.Name, "branch", cb.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase and/or jenkinsfolder %v are/is unavailable", c.Name, jfn))
	}

	if c.Spec.Versioning.Type == util.VersioningTypeEDP && hasNewVersion(cb) {
		if err := h.ProcessNewVersion(cb); err != nil {
			h.SetFailedFields(cb, actionType, err.Error())
			return errors.Wrapf(err, "couldn't process new version for %v branch", cb.Name)
		}
	}

	if err := triggerFunc(cb); err != nil {
		h.SetFailedFields(cb, actionType, err.Error())
		return err
	}

	return handler.NextServeOrNil(h.Next, cb)
}

func (h TriggerJob) SetIntermediateSuccessFields(cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
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

	if err := h.Client.Status().Update(context.TODO(), cb); err != nil {
		if err := h.Client.Update(context.TODO(), cb); err != nil {
			return errors.Wrapf(err, "SetIntermediateSuccessFields failed for %v branch", cb.Name)
		}
	}
	return nil
}

func (h TriggerJob) SetFailedFields(cb *codebaseApi.CodebaseBranch, a codebaseApi.ActionType, message string) {
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
		return nil, errors.Wrapf(err, "failed to get jenkins folder %v", name)
	}
	return i, nil
}

func (h TriggerJob) ProcessNewVersion(b *codebaseApi.CodebaseBranch) error {
	if err := h.Service.ResetBranchBuildCounter(b); err != nil {
		return err
	}

	if err := h.Service.ResetBranchSuccessBuildCounter(b); err != nil {
		return err
	}

	return h.Service.AppendVersionToTheHistorySlice(b)
}

func hasNewVersion(b *codebaseApi.CodebaseBranch) bool {
	return !util.SearchVersion(b.Status.VersionHistory, *b.Spec.Version)
}

func isJenkinsFolderAvailable(jf *jenkinsApi.JenkinsFolder) bool {
	if jf == nil {
		return false
	}
	return jf.Status.Available
}
