package chain

import (
	"context"
	"fmt"

	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	maxPipelineRuns         = 500
	pipelineRunWarningLimit = 100
)

type ResolveStatus struct {
	client client.Client
}

func NewResolveStatus(k8sClient client.Client) *ResolveStatus {
	return &ResolveStatus{client: k8sClient}
}

func (r *ResolveStatus) ServeRequest(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	log := ctrl.LoggerFrom(ctx)

	if stageDeploy.Status.Status == codebaseApi.CDStageDeployStatusFailed {
		log.Info("CDStageDeploy has failed status. Retry to deploy.")
		return nil
	}

	if stageDeploy.Status.Status == codebaseApi.CDStageDeployStatusPending {
		log.Info("CDStageDeploy has pending status. Start deploying.")

		return nil
	}

	pipelineRun, err := r.getRunningPipelines(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to get running pipelines: %w", err)
	}

	if allPipelineRunsCompleted(pipelineRun.Items) {
		log.Info("All PipelineRuns have been completed.")

		stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusCompleted

		return nil
	}

	log.Info("Some PipelineRuns are still running.")

	return nil
}

func (r *ResolveStatus) getRunningPipelines(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) (*tektonpipelineApi.PipelineRunList, error) {
	log := ctrl.LoggerFrom(ctx)

	pipelineRun := &tektonpipelineApi.PipelineRunList{}

	if err := r.client.List(
		ctx,
		pipelineRun,
		client.InNamespace(stageDeploy.Namespace),
		client.MatchingLabels{
			"app.edp.epam.com/cdpipeline": stageDeploy.Spec.Pipeline,
			"app.edp.epam.com/cdstage":    stageDeploy.GetStageCRName(),
		},
		client.Limit(maxPipelineRuns),
	); err != nil {
		return nil, fmt.Errorf("failed to list PipelineRuns: %w", err)
	}

	if len(pipelineRun.Items) > pipelineRunWarningLimit {
		log.Info("Warning: too many PipelineRuns found. Consider to clean up old PipelineRuns.")
	}

	return pipelineRun, nil
}

func allPipelineRunsCompleted(pipelineRuns []tektonpipelineApi.PipelineRun) bool {
	for i := range pipelineRuns {
		if pipelineRuns[i].Status.CompletionTime == nil {
			return false
		}
	}

	return true
}
