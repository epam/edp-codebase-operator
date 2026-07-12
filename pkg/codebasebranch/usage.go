package codebasebranch

import (
	"context"
	"fmt"
	"slices"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// FindBranchUsage returns a human-readable description of the first deployment resource
// that references the given CodebaseBranch, or an empty string when the branch is unused.
//
// A branch participates in deployments in two ways:
//   - CDPipeline.Spec.InputDockerStreams lists CodebaseBranch resource names
//     (resolved to CodebaseImageStreams via the codebasebranch label);
//   - Stage.Spec.QualityGates references autotest codebase name + git branch name.
//
// Resources that are being deleted are not counted as usage.
func FindBranchUsage(ctx context.Context, c client.Client, branch *codebaseApi.CodebaseBranch) (string, error) {
	pipelines := &pipelineApi.CDPipelineList{}
	if err := c.List(ctx, pipelines, client.InNamespace(branch.Namespace)); err != nil {
		// The CD pipeline CRDs are owned by edp-cd-pipeline-operator, which may not be
		// installed alongside this operator; in that case nothing can reference the branch.
		if isKindUnavailable(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to list CDPipelines: %w", err)
	}

	for i := range pipelines.Items {
		pipeline := &pipelines.Items[i]

		if pipeline.DeletionTimestamp != nil {
			continue
		}

		if slices.Contains(pipeline.Spec.InputDockerStreams, branch.Name) {
			return fmt.Sprintf("CDPipeline %s (inputDockerStreams)", pipeline.Name), nil
		}
	}

	stages := &pipelineApi.StageList{}
	if err := c.List(ctx, stages, client.InNamespace(branch.Namespace)); err != nil {
		if isKindUnavailable(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to list Stages: %w", err)
	}

	for i := range stages.Items {
		stage := &stages.Items[i]

		if stage.DeletionTimestamp != nil {
			continue
		}

		for _, gate := range stage.Spec.QualityGates {
			if gate.AutotestName != nil && *gate.AutotestName == branch.Spec.CodebaseName &&
				gate.BranchName != nil && *gate.BranchName == branch.Spec.BranchName {
				return fmt.Sprintf("Stage %s of CDPipeline %s (autotest quality gate)", stage.Name, stage.Spec.CdPipeline), nil
			}
		}
	}

	return "", nil
}

func isKindUnavailable(err error) bool {
	return apimeta.IsNoMatchError(err) || runtime.IsNotRegisteredError(err)
}
