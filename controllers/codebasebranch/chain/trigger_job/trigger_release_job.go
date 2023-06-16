package trigger_job

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type TriggerReleaseJob struct {
	TriggerJob
}

func (h TriggerReleaseJob) ServeRequest(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	return h.Trigger(ctx, cb, codebaseApi.TriggerReleaseJob, h.Service.TriggerReleaseJob)
}
