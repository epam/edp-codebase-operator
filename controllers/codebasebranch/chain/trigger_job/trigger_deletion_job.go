package trigger_job

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type TriggerDeletionJob struct {
	TriggerJob
}

func (h TriggerDeletionJob) ServeRequest(cb *codebaseApi.CodebaseBranch) error {
	return h.Trigger(cb, codebaseApi.TriggerDeletionJob, h.Service.TriggerDeletionJob)
}
