package trigger_release_job

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	jfv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"time"
)

type TriggerReleaseJob struct {
	Next    handler.CodebaseBranchHandler
	Client  client.Client
	Service service.CodebaseBranchService
}

var log = logf.Log.WithName("trigger-release-job-chain")

func (h TriggerReleaseJob) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	if err := h.setIntermediateSuccessFields(cb, edpv1alpha1.AcceptCodebaseBranchRegistration); err != nil {
		return err
	}

	c, err := util.GetCodebase(h.Client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		setFailedFields(cb, v1alpha1.TriggerReleaseJob, err.Error())
		return err
	}

	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jf, err := h.getJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		setFailedFields(cb, v1alpha1.TriggerReleaseJob, err.Error())
		return err
	}

	if !c.Status.Available && isJenkinsFolderAvailable(jf) {
		log.Info("couldn't start reconciling for branch. someone of codebase or jenkins folder is unavailable",
			"codebase", c.Name, "branch", cb.Name)
		return util.NewCodebaseBranchReconcileError(fmt.Sprintf("%v codebase is unavailable", c.Name))
	}

	if c.Spec.Versioning.Type == util.VersioningTypeEDP && hasNewVersion(cb) {
		if err := h.processNewVersion(cb); err != nil {
			setFailedFields(cb, v1alpha1.TriggerReleaseJob, err.Error())
			return errors.Wrapf(err, "couldn't process new version for %v branch", cb.Name)
		}
	}

	if err := h.Service.TriggerReleaseJob(cb); err != nil {
		setFailedFields(cb, v1alpha1.TriggerReleaseJob, err.Error())
		return err
	}

	return handler.NextServeOrNil(h.Next, cb)
}

func (h TriggerReleaseJob) setIntermediateSuccessFields(cb *v1alpha1.CodebaseBranch, action edpv1alpha1.ActionType) error {
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
	}

	if err := h.Client.Status().Update(context.TODO(), cb); err != nil {
		if err := h.Client.Update(context.TODO(), cb); err != nil {
			return err
		}
	}
	return nil
}

func setFailedFields(cb *edpv1alpha1.CodebaseBranch, a edpv1alpha1.ActionType, message string) {
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
	}
}

func hasNewVersion(b *v1alpha1.CodebaseBranch) bool {
	return !util.SearchVersion(b.Status.VersionHistory, *b.Spec.Version)
}

func (h TriggerReleaseJob) processNewVersion(b *v1alpha1.CodebaseBranch) error {
	if err := h.Service.ResetBranchBuildCounter(b); err != nil {
		return err
	}

	if err := h.Service.ResetBranchSuccessBuildCounter(b); err != nil {
		return err
	}

	return h.Service.AppendVersionToTheHistorySlice(b)
}

func isJenkinsFolderAvailable(jf *jfv1alpha1.JenkinsFolder) bool {
	return jf == nil || !jf.Status.Available
}

func (h TriggerReleaseJob) getJenkinsFolder(name, namespace string) (*jfv1alpha1.JenkinsFolder, error) {
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
