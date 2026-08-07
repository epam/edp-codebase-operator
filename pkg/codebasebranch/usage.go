package codebasebranch

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/deploymentusage"
)

// autotestGate identifies the autotest a Stage quality gate runs: a codebase and one of
// its git branches. Stage.Spec.QualityGates references both, never the CodebaseBranch
// resource name.
type autotestGate struct {
	codebaseName string
	branchName   string
}

// BranchUsageIndex answers "which deployment resources reference this CodebaseBranch"
// for many branches from a single read of the CDPipelines and Stages, which is what the
// stale branch sweep needs: it asks per branch while the answer set is the same for all.
type BranchUsageIndex struct {
	// byStreamName is keyed by CodebaseBranch resource name, as held in
	// CDPipeline.Spec.InputDockerStreams.
	byStreamName map[string][]deploymentusage.Reference
	byAutotest   map[autotestGate][]deploymentusage.Reference
}

// NewBranchUsageIndex reads the deployment resources of the namespace once and indexes
// the references they hold. Resources that are being deleted are not counted as usage.
//
// The snapshot is never refreshed, so callers that must not act on stale data build one
// per request.
func NewBranchUsageIndex(ctx context.Context, c client.Client, namespace string) (*BranchUsageIndex, error) {
	index := &BranchUsageIndex{
		byStreamName: make(map[string][]deploymentusage.Reference),
		byAutotest:   make(map[autotestGate][]deploymentusage.Reference),
	}

	pipelines, err := deploymentusage.ListActiveCDPipelines(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	for i := range pipelines {
		index.addPipeline(&pipelines[i])
	}

	gates, err := deploymentusage.FindAutotestGates(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	for _, gate := range gates {
		key := autotestGate{codebaseName: gate.AutotestName, branchName: gate.BranchName}
		index.byAutotest[key] = append(index.byAutotest[key], gate.Reference)
	}

	return index, nil
}

func (i *BranchUsageIndex) addPipeline(pipeline *pipelineApi.CDPipeline) {
	// A pipeline listing the same stream twice must still be reported once.
	seen := make(map[string]struct{}, len(pipeline.Spec.InputDockerStreams))

	for _, stream := range pipeline.Spec.InputDockerStreams {
		if _, duplicate := seen[stream]; duplicate {
			continue
		}

		seen[stream] = struct{}{}

		i.byStreamName[stream] = append(i.byStreamName[stream], deploymentusage.Reference{
			Kind:   deploymentusage.KindCDPipeline,
			Name:   pipeline.Name,
			Field:  deploymentusage.FieldInputDockerStreams,
			Reason: "inputDockerStreams",
		})
	}
}

// Find returns every deployment resource that references the given CodebaseBranch, or an
// empty slice when the branch is unused.
//
// A branch participates in deployments in two ways:
//   - CDPipeline.Spec.InputDockerStreams lists CodebaseBranch resource names
//     (resolved to CodebaseImageStreams via the codebasebranch label);
//   - Stage.Spec.QualityGates references autotest codebase name + git branch name.
func (i *BranchUsageIndex) Find(branch *codebaseApi.CodebaseBranch) []deploymentusage.Reference {
	pipelineRefs := i.byStreamName[branch.Name]
	gateRefs := i.byAutotest[autotestGate{
		codebaseName: branch.Spec.CodebaseName,
		branchName:   branch.Spec.BranchName,
	}]

	if len(pipelineRefs) == 0 && len(gateRefs) == 0 {
		return nil
	}

	// A fresh slice: appending to an indexed one writes into the index's own storage.
	refs := make([]deploymentusage.Reference, 0, len(pipelineRefs)+len(gateRefs))
	refs = append(refs, pipelineRefs...)

	return append(refs, gateRefs...)
}

// FindBranchUsage returns every deployment resource that references the given
// CodebaseBranch, or an empty slice when the branch is unused. It reads the deployment
// resources on every call, as admission requires; callers checking many branches at once
// should build a BranchUsageIndex.
func FindBranchUsage(
	ctx context.Context,
	c client.Client,
	branch *codebaseApi.CodebaseBranch,
) ([]deploymentusage.Reference, error) {
	index, err := NewBranchUsageIndex(ctx, c, branch.Namespace)
	if err != nil {
		return nil, err
	}

	return index.Find(branch), nil
}
