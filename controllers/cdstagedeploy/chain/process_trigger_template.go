package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	tektonTriggersApi "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebaseimagestream"
)

var (
	errLasTagNotFound                = fmt.Errorf("last tag not found")
	errEmptyTriggerTemplateResources = fmt.Errorf("trigger template resources is empty")
)

type TriggerTemplateApplicationPayload struct {
	ImageTag string `json:"imageTag"`
}

type ProcessTriggerTemplate struct {
	k8sClient client.Client
}

func NewProcessTriggerTemplate(k8sClient client.Client) *ProcessTriggerTemplate {
	return &ProcessTriggerTemplate{k8sClient: k8sClient}
}

func (h *ProcessTriggerTemplate) ServeRequest(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	log := ctrl.LoggerFrom(ctx).WithValues("stage", stageDeploy.Spec.Stage, "pipeline", stageDeploy.Spec.Pipeline, "status", stageDeploy.Status.Status)

	if skipPipelineCreation(stageDeploy) {
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

	appPayload, err := h.getAppPayload(ctx, pipeline)
	if err != nil {
		if errors.Is(err, errLasTagNotFound) {
			stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusCompleted
			return nil
		}

		return err
	}

	stage := &pipelineAPi.Stage{}
	if err = h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: stageDeploy.Namespace,
		Name:      stageDeploy.GetStageCRName(),
	}, stage); err != nil {
		return fmt.Errorf("failed to get Stage: %w", err)
	}

	rawResource, err := h.getRawTriggerTemplateResource(ctx, stage.Spec.TriggerTemplate, stageDeploy.Namespace)
	if err != nil {
		if errors.Is(err, errEmptyTriggerTemplateResources) {
			log.Info("No resource templates found in the trigger template. Skip processing.", "triggertemplate", stage.Spec.TriggerType)

			stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusCompleted

			return nil
		}

		return err
	}

	if err = h.createTriggerTemplateResource(
		ctx,
		stageDeploy.Namespace,
		rawResource,
		appPayload,
		[]byte(stage.Spec.Name),
		[]byte(pipeline.Spec.Name),
		[]byte(stage.Spec.ClusterName),
	); err != nil {
		return err
	}

	stageDeploy.Status.Status = codebaseApi.CDStageDeployStatusRunning

	log.Info("TriggerTemplate for auto-deploy has been processed successfully.")

	return nil
}

func skipPipelineCreation(stageDeploy *codebaseApi.CDStageDeploy) bool {
	if stageDeploy.Status.Status != codebaseApi.CDStageDeployStatusFailed &&
		stageDeploy.Status.Status != codebaseApi.CDStageDeployStatusPending {
		return true
	}

	return false
}

func (h *ProcessTriggerTemplate) getAppPayload(ctx context.Context, pipeline *pipelineAPi.CDPipeline) (json.RawMessage, error) {
	log := ctrl.LoggerFrom(ctx)

	appPayload := make(map[string]TriggerTemplateApplicationPayload, len(pipeline.Spec.InputDockerStreams))

	for _, stream := range pipeline.Spec.InputDockerStreams {
		imageStreamName := put_codebase_image_stream.ProcessNameToK8sConvention(stream)

		imageStream := &codebaseApi.CodebaseImageStream{}
		if err := h.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: pipeline.Namespace,
			Name:      put_codebase_image_stream.ProcessNameToK8sConvention(imageStreamName),
		}, imageStream); err != nil {
			return nil, fmt.Errorf("failed to get %s CodebaseImageStream: %w", imageStreamName, err)
		}

		tag, err := codebaseimagestream.GetLastTag(imageStream.Spec.Tags, log)
		if err != nil {
			log.Info("Codebase doesn't have tags in the CodebaseImageStream. Skip auto-deploy.", "codebase", imageStream.Spec.Codebase, "imagestream", imageStreamName)

			return nil, errLasTagNotFound
		}

		appPayload[imageStream.Spec.Codebase] = TriggerTemplateApplicationPayload{
			ImageTag: tag.Name,
		}
	}

	rawAppPayload, err := json.Marshal(appPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application payload: %w", err)
	}

	return rawAppPayload, nil
}

func (h *ProcessTriggerTemplate) getRawTriggerTemplateResource(ctx context.Context, triggerTemplateName, ns string) ([]byte, error) {
	template := &tektonTriggersApi.TriggerTemplate{}
	if err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      triggerTemplateName,
	}, template); err != nil {
		return nil, fmt.Errorf("failed to get TriggerTemplate: %w", err)
	}

	if len(template.Spec.ResourceTemplates) == 0 {
		return nil, errEmptyTriggerTemplateResources
	}

	rawPipeRun := make([]byte, len(template.Spec.ResourceTemplates[0].RawExtension.Raw))
	copy(rawPipeRun, template.Spec.ResourceTemplates[0].RawExtension.Raw)

	return rawPipeRun, nil
}

func (h *ProcessTriggerTemplate) createTriggerTemplateResource(
	ctx context.Context,
	ns string,
	rawPipeRun,
	appPayload,
	stage,
	pipeline,
	clusterSecret []byte,
) error {
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.APPLICATIONS_PAYLOAD)"), bytes.ReplaceAll(appPayload, []byte(`"`), []byte(`\"`)))
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.CDSTAGE)"), stage)
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.CDPIPELINE)"), pipeline)
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.KUBECONFIG_SECRET_NAME)"), clusterSecret)

	data := &unstructured.Unstructured{}
	if err := data.UnmarshalJSON(rawPipeRun); err != nil {
		return fmt.Errorf("couldn't unmarshal json from the TriggerTemplate: %w", err)
	}

	data.SetNamespace(ns)

	if err := h.k8sClient.Create(ctx, data); err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	return nil
}
