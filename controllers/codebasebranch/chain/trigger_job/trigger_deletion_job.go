package trigger_job

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type TriggerDeletionJob struct {
	TriggerJob
}

func (h TriggerDeletionJob) ServeRequest(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	return h.Trigger(ctx, cb, codebaseApi.TriggerDeletionJob, h.Service.TriggerDeletionJob)
}
