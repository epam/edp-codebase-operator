package trigger_job

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type TriggerDeletionJob struct {
	TriggerJob
}

func (h TriggerDeletionJob) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	return h.Trigger(cb, v1alpha1.TriggerDeletionJob, h.Service.TriggerDeletionJob)
}
