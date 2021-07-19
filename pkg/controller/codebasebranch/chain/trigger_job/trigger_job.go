package trigger_job

import (
	"context"
	"fmt"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jfv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("trigger-job-chain")

type TriggerJob struct {
	Client  client.Client
	Service service.CodebaseBranchService
	Next    handler.CodebaseBranchHandler
}

func (h TriggerJob) Trigger(cb *v1alpha1.CodebaseBranch, actionType edpv1alpha1.ActionType,
	triggerFunc func(cb *v1alpha1.CodebaseBranch) error) error {
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

	if !c.Status.Available && isJenkinsFolderAvailable(jf) {
		log.Info("couldn't start reconciling for branch. someone of codebase or jenkins folder is unavailable",
			"codebase", c.Name, "branch", cb.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase is unavailable", c.Name))
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

func (h TriggerJob) SetIntermediateSuccessFields(cb *v1alpha1.CodebaseBranch, action edpv1alpha1.ActionType) error {
	cb.Status = v1alpha1.CodebaseBranchStatus{
		Status:              model.StatusInit,
		LastTimeUpdated:     time.Now(),
		Action:              action,
		Result:              edpv1alpha1.Success,
		Username:            "system",
		Value:               "inactive",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
		FailureCount:        cb.Status.FailureCount,
	}

	if err := h.Client.Status().Update(context.TODO(), cb); err != nil {
		if err := h.Client.Update(context.TODO(), cb); err != nil {
			return err
		}
	}
	return nil
}

func (h TriggerJob) SetFailedFields(cb *edpv1alpha1.CodebaseBranch, a edpv1alpha1.ActionType, message string) {
	cb.Status = edpv1alpha1.CodebaseBranchStatus{
		Status:              util.StatusFailed,
		LastTimeUpdated:     time.Now(),
		Username:            "system",
		Action:              a,
		Result:              edpv1alpha1.Error,
		DetailedMessage:     message,
		Value:               "failed",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
		FailureCount:        cb.Status.FailureCount,
	}
}

func (h TriggerJob) GetJenkinsFolder(name, namespace string) (*jfv1alpha1.JenkinsFolder, error) {
	i := &jfv1alpha1.JenkinsFolder{}
	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, i)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get jenkins folder %v", name)
	}
	return i, nil
}

func (h TriggerJob) ProcessNewVersion(b *v1alpha1.CodebaseBranch) error {
	if err := h.Service.ResetBranchBuildCounter(b); err != nil {
		return err
	}

	if err := h.Service.ResetBranchSuccessBuildCounter(b); err != nil {
		return err
	}

	return h.Service.AppendVersionToTheHistorySlice(b)
}

func hasNewVersion(b *v1alpha1.CodebaseBranch) bool {
	return !util.SearchVersion(b.Status.VersionHistory, *b.Spec.Version)
}

func isJenkinsFolderAvailable(jf *jfv1alpha1.JenkinsFolder) bool {
	return jf == nil || !jf.Status.Available
}
