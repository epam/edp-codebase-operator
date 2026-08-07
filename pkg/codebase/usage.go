package codebase

import (
	"context"
	"fmt"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/deploymentusage"
)

// FindCodebaseUsage returns every deployment resource that references the
// given Codebase, or an empty slice when the codebase is unused.
//
// A codebase participates in deployments in three ways:
//   - CDPipeline.Spec.Applications (or ApplicationsToPromote) lists Codebase names
//     (application components);
//   - CDPipeline.Spec.InputDockerStreams lists the names of the codebase's branches;
//   - Stage.Spec.QualityGates references a Codebase name via AutotestName
//     (autotest components).
//
// Resources that are being deleted are not counted as usage.
func FindCodebaseUsage(
	ctx context.Context,
	c client.Client,
	codebase *codebaseApi.Codebase,
) ([]deploymentusage.Reference, error) {
	var refs []deploymentusage.Reference

	pipelines, err := deploymentusage.ListActiveCDPipelines(ctx, c, codebase.Namespace)
	if err != nil {
		return nil, err
	}

	branchNames, err := listBranchNames(ctx, c, codebase)
	if err != nil {
		return nil, err
	}

	for i := range pipelines {
		pipeline := &pipelines[i]

		field, reason, ok := matchPipeline(pipeline, codebase.Name, branchNames)
		if !ok {
			continue
		}

		refs = append(refs, deploymentusage.Reference{
			Kind:   deploymentusage.KindCDPipeline,
			Name:   pipeline.Name,
			Field:  field,
			Reason: reason,
		})
	}

	gates, err := deploymentusage.FindAutotestGates(ctx, c, codebase.Namespace)
	if err != nil {
		return nil, err
	}

	// A codebase is the autotest itself, so the gate's branch is not part of the match.
	for _, gate := range gates {
		if gate.AutotestName == codebase.Name {
			refs = append(refs, gate.Reference)
		}
	}

	return refs, nil
}

// matchPipeline reports how the pipeline references the codebase, if it does.
//
// applicationsToPromote is normally a subset of applications, but nothing enforces that,
// so both are checked. inputDockerStreams holds branch names rather than the codebase
// name: deleting the codebase cascades to its branches, and the CodebaseBranch webhook
// lets that cascade through, so a pipeline consuming any of them blocks the codebase too.
// A pipeline is reported once, naming the field it was matched on.
func matchPipeline(
	pipeline *pipelineApi.CDPipeline,
	codebaseName string,
	branchNames map[string]struct{},
) (field, reason string, ok bool) {
	switch {
	case slices.Contains(pipeline.Spec.Applications, codebaseName):
		return deploymentusage.FieldApplications, "applications", true
	case slices.Contains(pipeline.Spec.ApplicationsToPromote, codebaseName):
		return deploymentusage.FieldApplicationsToPromote, "applicationsToPromote", true
	}

	for _, stream := range pipeline.Spec.InputDockerStreams {
		if _, isBranch := branchNames[stream]; isBranch {
			return deploymentusage.FieldInputDockerStreams,
				fmt.Sprintf("branch %s in inputDockerStreams", stream),
				true
		}
	}

	return "", "", false
}

// listBranchNames returns the resource names of the codebase's branches, which is what
// CDPipeline.Spec.InputDockerStreams holds.
func listBranchNames(
	ctx context.Context,
	c client.Client,
	codebase *codebaseApi.Codebase,
) (map[string]struct{}, error) {
	branches := &codebaseApi.CodebaseBranchList{}
	if err := c.List(
		ctx,
		branches,
		client.InNamespace(codebase.Namespace),
		client.MatchingFields{BranchCodebaseNameIndex: codebase.Name},
	); err != nil {
		// Without the CodebaseBranch kind there are no branches, so nothing can reference
		// one. Any other failure means usage could not be verified and must surface.
		if deploymentusage.IsKindUnavailable(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list CodebaseBranches: %w", err)
	}

	names := make(map[string]struct{}, len(branches.Items))

	for i := range branches.Items {
		names[branches.Items[i].Name] = struct{}{}
	}

	return names, nil
}
