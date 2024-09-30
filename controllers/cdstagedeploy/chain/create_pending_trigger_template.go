package chain

import (
	"context"
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/autodeploy"
	"github.com/epam/edp-codebase-operator/v2/pkg/tektoncd"
)

type CreatePendingTriggerTemplate struct {
	k8sClient                 client.Client
	triggerTemplateManager    tektoncd.TriggerTemplateManager
	autoDeployStrategyManager autodeploy.Manager
}

func NewCreatePendingTriggerTemplate(k8sClient client.Client, triggerTemplateManager tektoncd.TriggerTemplateManager, autoDeployStrategyManager autodeploy.Manager) *CreatePendingTriggerTemplate {
	return &CreatePendingTriggerTemplate{k8sClient: k8sClient, triggerTemplateManager: triggerTemplateManager, autoDeployStrategyManager: autoDeployStrategyManager}
}

func (h *CreatePendingTriggerTemplate) ServeRequest(
	ctx context.Context,
	stageDeploy *codebaseApi.CDStageDeploy,
) error {
	log := ctrl.LoggerFrom(ctx).WithValues("stage", stageDeploy.Spec.Stage, "pipeline", stageDeploy.Spec.Pipeline, "status", stageDeploy.Status.Status)

	if skipPipelineRunCreation(stageDeploy) {
		log.Info("Skip processing TriggerTemplate for auto-deploy.")

		return nil
	}

	log.Info("Start processing TriggerTemplate for auto-deploy.")

	pipeline := &pipelineAPi.CDPipeline{}
	if err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: stageDeploy.Namespace,
		Name:      stageDeploy.Spec.Pipeline,
	}, pipeline); err != nil {
		return fmt.Errorf("failed to get CDPipeline: %w", err)
	}

	stage := &pipelineAPi.Stage{}
	if err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: stageDeploy.Namespace,
		Name:      stageDeploy.GetStageCRName(),
	}, stage); err != nil {
		return fmt.Errorf("failed to get Stage: %w", err)
	}

	rawResource, err := h.triggerTemplateManager.GetRawResourceFromTriggerTemplate(ctx, stage.Spec.TriggerTemplate, stageDeploy.Namespace)
	if err != nil {
		if errors.Is(err, tektoncd.ErrEmptyTriggerTemplateResources) {
			log.Info("No resource templates found in the trigger template. Skip processing.", "triggertemplate", stage.Spec.TriggerType)

			stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusCompleted

			return nil
		}

		return fmt.Errorf("failed to get raw resource from TriggerTemplate: %w", err)
	}

	appPayload, err := h.autoDeployStrategyManager.GetAppPayloadForCurrentWithStableStrategy(
		ctx,
		stageDeploy.Spec.Tag,
		pipeline,
		stage,
	)
	if err != nil {
		if errors.Is(err, autodeploy.ErrLasTagNotFound) {
			stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusCompleted
			return nil
		}

		return fmt.Errorf("failed to get app payload: %w", err)
	}

	if err = h.triggerTemplateManager.CreatePendingPipelineRun(
		ctx,
		stageDeploy.Namespace,
		stageDeploy.Name,
		rawResource,
		appPayload,
		[]byte(stage.Spec.Name),
		[]byte(pipeline.Spec.Name),
		[]byte(stage.Spec.ClusterName),
	); err != nil {
		return fmt.Errorf("failed to create PendingPipelineRun: %w", err)
	}

	stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusInQueue

	log.Info("TriggerTemplate for auto-deploy has been processed successfully.")

	return nil
}
