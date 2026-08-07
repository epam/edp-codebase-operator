// Package deploymentusage provides shared helpers for detecting whether a
// Codebase or CodebaseBranch is still referenced by CD pipeline deployment
// resources (CDPipeline, Stage). It backs the admission webhooks that block
// deletion of resources that are still in use.
package deploymentusage

import (
	"context"
	"fmt"
	"strings"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

// Kinds of the deployment resources that can hold a reference. Reference.String
// branches on these, so they are not free-form.
const (
	KindCDPipeline = "CDPipeline"
	KindStage      = "Stage"
)

// Spec paths reported as Reference.Field, and in turn as metav1.StatusCause.Field.
// Clients key off these instead of parsing the message.
const (
	FieldApplications          = "spec.applications"
	FieldApplicationsToPromote = "spec.applicationsToPromote"
	FieldInputDockerStreams    = "spec.inputDockerStreams"
	FieldQualityGateAutotest   = "spec.qualityGates.autotestName"
)

// Reference describes a single deployment resource that references a
// Codebase or CodebaseBranch, blocking its deletion.
type Reference struct {
	Kind string
	Name string
	// Field is the JSON path holding the reference, e.g. "spec.inputDockerStreams".
	// It is surfaced as the Field of a metav1.StatusCause.
	Field string
	// Reason is a human-readable phrase, e.g. "inputDockerStreams" or
	// "autotest quality gate".
	Reason string
	// ParentCDPipeline is set only when Kind is KindStage.
	ParentCDPipeline string
}

// String renders the reference as it appears in the denial message shown to users.
func (r Reference) String() string {
	if r.Kind == KindStage {
		return fmt.Sprintf("%s %s of %s %s (%s)", KindStage, r.Name, KindCDPipeline, r.ParentCDPipeline, r.Reason)
	}

	return fmt.Sprintf("%s %s (%s)", r.Kind, r.Name, r.Reason)
}

// Join renders references as a single description, in the form used by both the
// admission denial message and the stale-branch retention verdict.
func Join(refs []Reference) string {
	descriptions := make([]string, 0, len(refs))

	for _, ref := range refs {
		descriptions = append(descriptions, ref.String())
	}

	return strings.Join(descriptions, "; ")
}

// ListActiveCDPipelines returns the CDPipelines in the given namespace that
// are not being deleted. When the CD pipeline CRDs are not installed in the
// cluster, it returns an empty, non-error result.
func ListActiveCDPipelines(ctx context.Context, c client.Client, namespace string) ([]pipelineApi.CDPipeline, error) {
	pipelines := &pipelineApi.CDPipelineList{}
	if err := c.List(ctx, pipelines, client.InNamespace(namespace)); err != nil {
		if IsKindUnavailable(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list CDPipelines: %w", err)
	}

	active := make([]pipelineApi.CDPipeline, 0, len(pipelines.Items))

	for i := range pipelines.Items {
		if pipelines.Items[i].DeletionTimestamp != nil {
			continue
		}

		active = append(active, pipelines.Items[i])
	}

	return active, nil
}

// AutotestGate is a Stage quality gate that runs an autotest, together with the Reference
// describing the Stage holding it. Codebase and CodebaseBranch are both referenced through
// this one gate field and key on different parts of it.
type AutotestGate struct {
	Reference

	// AutotestName is the referenced Codebase name; never empty.
	AutotestName string
	// BranchName is the referenced git branch name, empty when the gate leaves it unset.
	BranchName string
}

// FindAutotestGates returns every autotest quality gate of the namespace. Manual gates
// leave AutotestName unset and are dropped here, so no caller can mistake one for a
// match by comparing unset fields.
func FindAutotestGates(ctx context.Context, c client.Client, namespace string) ([]AutotestGate, error) {
	stages, err := listActiveStages(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	var gates []AutotestGate

	for i := range stages {
		stage := &stages[i]

		for _, gate := range stage.Spec.QualityGates {
			if gate.AutotestName == nil || *gate.AutotestName == "" {
				continue
			}

			gates = append(gates, AutotestGate{
				Reference: Reference{
					Kind:             KindStage,
					Name:             stage.Name,
					Field:            FieldQualityGateAutotest,
					Reason:           "autotest quality gate",
					ParentCDPipeline: stage.Spec.CdPipeline,
				},
				AutotestName: *gate.AutotestName,
				BranchName:   ptr.Deref(gate.BranchName, ""),
			})
		}
	}

	return gates, nil
}

// listActiveStages returns the Stages in the given namespace that are not
// being deleted. When the CD pipeline CRDs are not installed in the cluster,
// it returns an empty, non-error result.
func listActiveStages(ctx context.Context, c client.Client, namespace string) ([]pipelineApi.Stage, error) {
	stages := &pipelineApi.StageList{}
	if err := c.List(ctx, stages, client.InNamespace(namespace)); err != nil {
		if IsKindUnavailable(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list Stages: %w", err)
	}

	active := make([]pipelineApi.Stage, 0, len(stages.Items))

	for i := range stages.Items {
		if stages.Items[i].DeletionTimestamp != nil {
			continue
		}

		active = append(active, stages.Items[i])
	}

	return active, nil
}

// IsKindUnavailable reports whether err indicates that a CRD kind is not
// registered/installed in the cluster (e.g. edp-cd-pipeline-operator is not
// deployed alongside this operator).
func IsKindUnavailable(err error) bool {
	return apimeta.IsNoMatchError(err) || runtime.IsNotRegisteredError(err)
}
