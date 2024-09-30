package tektoncd

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonTriggersApi "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

var ErrEmptyTriggerTemplateResources = fmt.Errorf("trigger template resources is empty")

type TriggerTemplateManager interface {
	GetRawResourceFromTriggerTemplate(ctx context.Context, triggerTemplateName, ns string) ([]byte, error)
	CreatePipelineRun(ctx context.Context, ns, cdStageDeployName string, rawPipeRun, appPayload, stage, pipeline, clusterSecret []byte) error
	CreatePendingPipelineRun(ctx context.Context, ns, cdStageDeployName string, rawPipeRun, appPayload, stage, pipeline, clusterSecret []byte) error
}

var _ TriggerTemplateManager = &TektonTriggerTemplateManager{}

type TektonTriggerTemplateManager struct {
	k8sClient client.Client
}

func NewTektonTriggerTemplateManager(k8sClient client.Client) *TektonTriggerTemplateManager {
	return &TektonTriggerTemplateManager{k8sClient: k8sClient}
}

func (h *TektonTriggerTemplateManager) GetRawResourceFromTriggerTemplate(ctx context.Context, triggerTemplateName, ns string) ([]byte, error) {
	template := &tektonTriggersApi.TriggerTemplate{}
	if err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      triggerTemplateName,
	}, template); err != nil {
		return nil, fmt.Errorf("failed to get TriggerTemplate: %w", err)
	}

	if len(template.Spec.ResourceTemplates) == 0 {
		return nil, ErrEmptyTriggerTemplateResources
	}

	rawPipeRun := make([]byte, len(template.Spec.ResourceTemplates[0].RawExtension.Raw))
	copy(rawPipeRun, template.Spec.ResourceTemplates[0].RawExtension.Raw)

	return rawPipeRun, nil
}

func (h *TektonTriggerTemplateManager) CreatePipelineRun(
	ctx context.Context,
	ns,
	cdStageDeployName string,
	rawPipeRun,
	appPayload,
	stage,
	pipeline,
	clusterSecret []byte,
) error {
	data, err := makeUnstructuredPipelineRun(ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)
	if err != nil {
		return err
	}

	if err = h.k8sClient.Create(ctx, data); err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	return nil
}

func (h *TektonTriggerTemplateManager) CreatePendingPipelineRun(
	ctx context.Context,
	ns,
	cdStageDeployName string,
	rawPipeRun,
	appPayload,
	stage,
	pipeline,
	clusterSecret []byte,
) error {
	data, err := makeUnstructuredPipelineRun(ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)
	if err != nil {
		return err
	}

	spec, ok := data.Object["spec"].(map[string]interface{})
	if !ok {
		return errors.New("invalid PipelineRun spec")
	}

	spec["status"] = tektonpipelineApi.PipelineRunSpecStatusPending

	if err = h.k8sClient.Create(ctx, data); err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	return nil
}

func makeUnstructuredPipelineRun(
	ns,
	cdStageDeployName string,
	rawPipeRun []byte,
	appPayload []byte,
	stage []byte,
	pipeline []byte,
	clusterSecret []byte,
) (*unstructured.Unstructured, error) {
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.APPLICATIONS_PAYLOAD)"), bytes.ReplaceAll(appPayload, []byte(`"`), []byte(`\"`)))
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.CDSTAGE)"), stage)
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.CDPIPELINE)"), pipeline)
	rawPipeRun = bytes.ReplaceAll(rawPipeRun, []byte("$(tt.params.KUBECONFIG_SECRET_NAME)"), clusterSecret)

	data := &unstructured.Unstructured{}
	if err := data.UnmarshalJSON(rawPipeRun); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal json from the TriggerTemplate: %w", err)
	}

	data.SetNamespace(ns)

	labels := data.GetLabels()
	labels[codebaseApi.CdStageDeployLabel] = cdStageDeployName
	data.SetLabels(labels)

	return data, nil
}
