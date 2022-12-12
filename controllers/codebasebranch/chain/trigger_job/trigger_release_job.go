package trigger_job

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type TriggerReleaseJob struct {
	TriggerJob
}

func (h TriggerReleaseJob) ServeRequest(cb *codebaseApi.CodebaseBranch) error {
	return h.Trigger(cb, codebaseApi.TriggerReleaseJob, h.Service.TriggerReleaseJob)
}
