package chain

import (
	"context"
	"fmt"

	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type ProcessPendingPipeRuns struct {
	k8sClient client.Client
}

func NewProcessPendingPipeRuns(k8sClient client.Client) *ProcessPendingPipeRuns {
	return &ProcessPendingPipeRuns{k8sClient: k8sClient}
}

func (r *ProcessPendingPipeRuns) ServeRequest(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	log := ctrl.LoggerFrom(ctx).WithValues("stage", stageDeploy.Spec.Stage, "pipeline", stageDeploy.Spec.Pipeline, "status", stageDeploy.Status.Status)

	if skipProcessingPendingPipeRuns(stageDeploy) {
		log.Info("Skip processing pending PipelineRuns for CDStageDeploy.")

		return nil
	}

	pipeRuns, err := r.getPipelines(ctx, stageDeploy)
	if err != nil {
		log.Info("Failed to get PipelineRuns.", "error", err)

		return nil
	}

	currentRun, err := getCdStageDeployPipeRun(pipeRuns, stageDeploy.Name)
	if err != nil {
		log.Info("Failed to get PipelineRun related to CDStageDeploy.", "error", err)
		return nil
	}

	if currentRun.IsDone() {
		stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusCompleted

		log.Info("PipelineRun is done. CdStageDeploy is completed.")

		return nil
	}

	if currentRun.Spec.Status == tektonpipelineApi.PipelineRunSpecStatusPending {
		if shouldStartPipeRun(currentRun.Name, pipeRuns) {
			if err = r.startPipelineRun(ctx, currentRun); err != nil {
				log.Info("Failed to start PipelineRun.", "error", err)

				return nil
			}

			stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusRunning
		}

		return nil
	}

	return nil
}

func skipProcessingPendingPipeRuns(stageDeploy *codebaseApi.CDStageDeploy) bool {
	if stageDeploy.IsInQueue() || stageDeploy.IsRunning() {
		return false
	}

	return true
}

func (r *ProcessPendingPipeRuns) startPipelineRun(ctx context.Context, pipeRun *tektonpipelineApi.PipelineRun) error {
	patch := []byte(`{"spec":{"status":""}}`)

	if err := r.k8sClient.Patch(ctx, pipeRun, client.RawPatch(types.MergePatchType, patch)); err != nil {
		return fmt.Errorf("failed to start PipelineRun: %w", err)
	}

	return nil
}

func (r *ProcessPendingPipeRuns) getPipelines(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) (*tektonpipelineApi.PipelineRunList, error) {
	log := ctrl.LoggerFrom(ctx)

	pipelineRun := &tektonpipelineApi.PipelineRunList{}

	if err := r.k8sClient.List(
		ctx,
		pipelineRun,
		client.InNamespace(stageDeploy.Namespace),
		client.MatchingLabels{
			codebaseApi.CdPipelineLabel: stageDeploy.Spec.Pipeline,
			codebaseApi.CdStageLabel:    stageDeploy.GetStageCRName(),
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

func getCdStageDeployPipeRun(runs *tektonpipelineApi.PipelineRunList, cdStageDeployName string) (*tektonpipelineApi.PipelineRun, error) {
	for i := range runs.Items {
		if runs.Items[i].Labels[codebaseApi.CdStageDeployLabel] == cdStageDeployName {
			return &runs.Items[i], nil
		}
	}

	return nil, fmt.Errorf("pipeline run for CDStageDeploy %v not found", cdStageDeployName)
}

func shouldStartPipeRun(pipeRunName string, runs *tektonpipelineApi.PipelineRunList) bool {
	if !allPipelineRunsCompletedExcept(runs.Items, pipeRunName) {
		return false
	}

	return isFirsPendingPipelineRun(runs, pipeRunName)
}

func allPipelineRunsCompletedExcept(pipelineRuns []tektonpipelineApi.PipelineRun, except string) bool {
	for i := range pipelineRuns {
		if pipelineRuns[i].Name == except || pipelineRuns[i].Spec.Status == tektonpipelineApi.PipelineRunSpecStatusPending {
			continue
		}

		if !pipelineRuns[i].IsDone() {
			return false
		}
	}

	return true
}

// isFirsPendingPipelineRun checks if the pending pipeline run is the first one in the chain based on the CreationTimestamp field.
func isFirsPendingPipelineRun(runs *tektonpipelineApi.PipelineRunList, pipeRunName string) bool {
	if len(runs.Items) == 0 {
		return false
	}

	var firstRun *tektonpipelineApi.PipelineRun

	for i := range runs.Items {
		if runs.Items[i].Spec.Status != tektonpipelineApi.PipelineRunSpecStatusPending {
			continue
		}

		if firstRun == nil || runs.Items[i].CreationTimestamp.Before(&firstRun.CreationTimestamp) {
			firstRun = &runs.Items[i]
		}
	}

	return firstRun != nil && firstRun.Name == pipeRunName
}
