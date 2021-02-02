package trigger_job

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type TriggerReleaseJob struct {
	TriggerJob
}

func (h TriggerReleaseJob) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	return h.Trigger(cb, v1alpha1.TriggerReleaseJob, h.Service.TriggerReleaseJob)
}
