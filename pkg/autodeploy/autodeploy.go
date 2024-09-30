package autodeploy

import (
	"context"
	"encoding/json"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebaseimagestream"
)

var ErrLasTagNotFound = fmt.Errorf("last tag not found")

type Manager interface {
	GetAppPayloadForAllLatestStrategy(ctx context.Context, pipeline *pipelineAPi.CDPipeline) (json.RawMessage, error)
	GetAppPayloadForCurrentWithStableStrategy(ctx context.Context, current codebaseApi.CodebaseTag, pipeline *pipelineAPi.CDPipeline, stage *pipelineAPi.Stage) (json.RawMessage, error)
}

var _ Manager = &StrategyManager{}

type StrategyManager struct {
	k8sClient client.Client
}

type ApplicationPayload struct {
	ImageTag string `json:"imageTag"`
}

func NewStrategyManager(k8sClient client.Client) *StrategyManager {
	return &StrategyManager{k8sClient: k8sClient}
}

func (h *StrategyManager) GetAppPayloadForAllLatestStrategy(ctx context.Context, pipeline *pipelineAPi.CDPipeline) (json.RawMessage, error) {
	appPayload := make(map[string]ApplicationPayload, len(pipeline.Spec.InputDockerStreams))

	for _, stream := range pipeline.Spec.InputDockerStreams {
		imageStreamName := put_codebase_image_stream.ProcessNameToK8sConvention(stream)

		codebase, tag, err := h.getLatestTag(ctx, imageStreamName, pipeline.Namespace)
		if err != nil {
			return nil, err
		}

		appPayload[codebase] = ApplicationPayload{
			ImageTag: tag,
		}
	}

	rawAppPayload, err := json.Marshal(appPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application payload: %w", err)
	}

	return rawAppPayload, nil
}

func (h *StrategyManager) GetAppPayloadForCurrentWithStableStrategy(
	ctx context.Context,
	current codebaseApi.CodebaseTag,
	pipeline *pipelineAPi.CDPipeline,
	stage *pipelineAPi.Stage,
) (json.RawMessage, error) {
	appPayload := make(map[string]ApplicationPayload, len(pipeline.Spec.InputDockerStreams))

	for _, app := range pipeline.Spec.Applications {
		t, ok := stage.GetAnnotations()[fmt.Sprintf("app.edp.epam.com/%s", app)]
		if ok {
			appPayload[app] = ApplicationPayload{
				ImageTag: t,
			}
		}
	}

	appPayload[current.Codebase] = ApplicationPayload{
		ImageTag: current.Tag,
	}

	for _, stream := range pipeline.Spec.InputDockerStreams {
		imageStreamName := put_codebase_image_stream.ProcessNameToK8sConvention(stream)

		codebase, tag, err := h.getLatestTag(ctx, imageStreamName, pipeline.Namespace)
		if err != nil {
			return nil, err
		}

		if _, ok := appPayload[codebase]; !ok {
			appPayload[codebase] = ApplicationPayload{
				ImageTag: tag,
			}
		}
	}

	rawAppPayload, err := json.Marshal(appPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application payload: %w", err)
	}

	return rawAppPayload, nil
}

func (h *StrategyManager) getLatestTag(ctx context.Context, imageStreamName, namespace string) (codebase, tag string, e error) {
	imageStream := &codebaseApi.CodebaseImageStream{}
	if err := h.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      imageStreamName,
	}, imageStream); err != nil {
		return "", "", fmt.Errorf("failed to get %s CodebaseImageStream: %w", imageStreamName, err)
	}

	t, err := codebaseimagestream.GetLastTag(imageStream.Spec.Tags, ctrl.LoggerFrom(ctx))
	if err != nil {
		return "", "", ErrLasTagNotFound
	}

	return imageStream.Spec.Codebase, t.Name, nil
}
