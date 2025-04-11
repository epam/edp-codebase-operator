package autodeploy

import (
	"context"
	"encoding/json"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
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
		imageStream, err := codebaseimagestream.GetCodebaseImageStreamByCodebaseBaseBranchName(
			ctx,
			h.k8sClient,
			stream,
			pipeline.Namespace,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get codebase image stream for stream %s: %w", stream, err)
		}

		codebase, tag, err := h.getLatestTag(ctx, imageStream)
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
		imageStream, err := codebaseimagestream.GetCodebaseImageStreamByCodebaseBaseBranchName(
			ctx,
			h.k8sClient,
			stream,
			pipeline.Namespace,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get codebase image stream for stream %s: %w", stream, err)
		}

		codebase, tag, err := h.getLatestTag(ctx, imageStream)
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

func (*StrategyManager) getLatestTag(ctx context.Context, imageStream *codebaseApi.CodebaseImageStream) (codebase, tag string, e error) {
	t, err := codebaseimagestream.GetLastTag(imageStream.Spec.Tags, ctrl.LoggerFrom(ctx))
	if err != nil {
		return "", "", ErrLasTagNotFound
	}

	return imageStream.Spec.Codebase, t.Name, nil
}
