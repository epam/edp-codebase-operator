package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	tektonTriggersApi "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
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
	log := ctrl.LoggerFrom(ctx).WithValues("stage", stageDeploy.Spec.Stage, "pipeline", stageDeploy.Spec.Pipeline)

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
			log.Info("No resource templates found in the trigger template %s. Skip processing.", stage.Spec.TriggerType)

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
	); err != nil {
		return err
	}

	log.Info("TriggerTemplate for auto-deploy has been processed successfully.")

	return nil
}

func (h *ProcessTriggerTemplate) getAppPayload(ctx context.Context, pipeline *pipelineAPi.CDPipeline) (json.RawMessage, error) {
	log := ctrl.LoggerFrom(ctx)

	imageStreams, err := getCodebaseImageStreamMap(pipeline)
	if err != nil {
		return nil, err
	}

	appPayload := make(map[string]TriggerTemplateApplicationPayload, len(imageStreams))

	for codebase, stream := range imageStreams {
		imageStream := &codebaseApi.CodebaseImageStream{}
		if err = h.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: pipeline.Namespace,
			Name:      stream,
		}, imageStream); err != nil {
			return nil, fmt.Errorf("failed to get %s CodebaseImageStream: %w", stream, err)
		}

		var tag codebaseApi.Tag

		if tag, err = codebaseimagestream.GetLastTag(imageStream.Spec.Tags, log); err != nil {
			log.Info("Codebase %s doesn't have tags in the CodebaseImageStream %s. Skip auto-deploy.", codebase, stream)

			return nil, errLasTagNotFound
		}

		appPayload[codebase] = TriggerTemplateApplicationPayload{
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

func (h *ProcessTriggerTemplate) createTriggerTemplateResource(ctx context.Context, ns string, rawPipeRun, appPayload, stage, pipeline []byte) error {
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.APPLICATIONS_PAYLOAD)"), bytes.ReplaceAll(appPayload, []byte(`"`), []byte(`\"`)))
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.CDSTAGE)"), stage)
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.CDPIPELINE)"), pipeline)

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

// getCodebaseImageStreamMap returns map of codebase name and image stream name.
func getCodebaseImageStreamMap(pipeline *pipelineAPi.CDPipeline) (map[string]string, error) {
	m := make(map[string]string, len(pipeline.Spec.InputDockerStreams))

	for _, stream := range pipeline.Spec.InputDockerStreams {
		i := strings.LastIndex(stream, "-")

		if i == -1 {
			return nil, fmt.Errorf("invalid image stream name %s", stream)
		}

		m[stream[:i]] = stream
	}

	return m, nil
}
