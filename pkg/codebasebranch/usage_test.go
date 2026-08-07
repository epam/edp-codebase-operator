package codebasebranch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/deploymentusage"
)

func usageScheme(t *testing.T, withPipelineAPI bool) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	if withPipelineAPI {
		require.NoError(t, pipelineApi.AddToScheme(scheme))
	}

	return scheme
}

func usageBranch() *codebaseApi.CodebaseBranch {
	return &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{Name: "app-feature", Namespace: "default"},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "app",
			BranchName:   "feature",
		},
	}
}

func TestFindBranchUsage_InputDockerStreams(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"other-main", "app-feature"},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "CDPipeline demo (inputDockerStreams)", refs[0].String())
}

func TestFindBranchUsage_AutotestQualityGate(t *testing.T) {
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("feature"),
			}},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(stage).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "Stage demo-dev of CDPipeline demo (autotest quality gate)", refs[0].String())
}

// A manual quality gate leaves AutotestName/BranchName unset, and must never be treated
// as a match just because the nil pointers happen to compare equal.
func TestFindBranchUsage_ManualQualityGateNotMatched(t *testing.T) {
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "manual",
				StepName:        "approve",
			}},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(stage).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindBranchUsage_MultipleReferences(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"app-feature"},
		},
	}
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("feature"),
			}},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline, stage).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	require.Len(t, refs, 2)

	descriptions := []string{refs[0].String(), refs[1].String()}
	assert.Contains(t, descriptions, "CDPipeline demo (inputDockerStreams)")
	assert.Contains(t, descriptions, "Stage demo-dev of CDPipeline demo (autotest quality gate)")
}

func TestFindBranchUsage_TerminatingPipelineIgnored(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "demo",
			Namespace:         "default",
			DeletionTimestamp: ptr.To(metav1.Now()),
			Finalizers:        []string{"keep"},
		},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"app-feature"},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindBranchUsage_Unused(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"other-main"},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindBranchUsage_PipelineAPINotInstalled(t *testing.T) {
	// CDPipeline/Stage kinds are absent from the scheme, emulating a cluster
	// without edp-cd-pipeline-operator: the branch must be reported as unused.
	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, false)).Build()

	refs, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// One snapshot must answer for many branches, which is the whole point of the index:
// the sweep asks per branch while the deployment resources are read once.
func TestBranchUsageIndex_AnswersManyBranchesFromOneRead(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"app-feature"},
		},
	}
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-qa", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("autotest"),
			}},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline, stage).Build()

	index, err := NewBranchUsageIndex(context.Background(), k8sClient, "default")
	require.NoError(t, err)

	usedByPipeline := index.Find(usageBranch())
	require.Len(t, usedByPipeline, 1)
	assert.Equal(t, "CDPipeline demo (inputDockerStreams)", usedByPipeline[0].String())

	autotestBranch := usageBranch()
	autotestBranch.Name = "app-autotest"
	autotestBranch.Spec.BranchName = "autotest"

	usedByGate := index.Find(autotestBranch)
	require.Len(t, usedByGate, 1)
	assert.Equal(t, "Stage demo-qa of CDPipeline demo (autotest quality gate)", usedByGate[0].String())

	unused := usageBranch()
	unused.Name = "app-orphan"
	unused.Spec.BranchName = "orphan"
	assert.Empty(t, index.Find(unused))
}

// Find must not hand out the slice the index stores. Three pipelines leave that slice
// with spare capacity, so appending the gate reference onto it writes into the index's
// own backing array, letting a later lookup overwrite a result the caller still holds.
func TestBranchUsageIndex_FindDoesNotAliasStoredReferences(t *testing.T) {
	pipelineNames := []string{"demo-a", "demo-b", "demo-c"}
	objects := make([]client.Object, 0, len(pipelineNames)+1)

	for _, name := range pipelineNames {
		pipeline := &pipelineApi.CDPipeline{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec:       pipelineApi.CDPipelineSpec{InputDockerStreams: []string{"app-feature"}},
		}
		objects = append(objects, pipeline)
	}

	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-qa", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("feature"),
			}},
		},
	}
	objects = append(objects, stage)

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(objects...).Build()

	index, err := NewBranchUsageIndex(context.Background(), k8sClient, "default")
	require.NoError(t, err)

	first := index.Find(usageBranch())
	require.Len(t, first, 4)

	sentinel := deploymentusage.Reference{Kind: "sentinel"}
	first[len(first)-1] = sentinel

	// A second lookup must not reach back into the slice the first caller is holding.
	index.Find(usageBranch())

	assert.Equal(t, sentinel, first[len(first)-1],
		"Find returned a slice backed by the index, so a later lookup corrupted an earlier result")
}

// A pipeline may list the same stream twice; it is still one blocking resource.
func TestBranchUsageIndex_DeduplicatesRepeatedStream(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"app-feature", "app-feature"},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline).Build()

	index, err := NewBranchUsageIndex(context.Background(), k8sClient, "default")
	require.NoError(t, err)

	assert.Len(t, index.Find(usageBranch()), 1)
}

// Without the CD pipeline CRDs nothing can reference a branch, and the index must stay
// usable instead of failing the caller.
func TestBranchUsageIndex_NoPipelineCRDs(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, false)).Build()

	index, err := NewBranchUsageIndex(context.Background(), k8sClient, "default")
	require.NoError(t, err)
	assert.Empty(t, index.Find(usageBranch()))
}
