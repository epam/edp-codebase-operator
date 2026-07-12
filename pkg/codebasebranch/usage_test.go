package codebasebranch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
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

	usage, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Equal(t, "CDPipeline demo (inputDockerStreams)", usage)
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

	usage, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Equal(t, "Stage demo-dev of CDPipeline demo (autotest quality gate)", usage)
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

	usage, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, usage)
}

func TestFindBranchUsage_Unused(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			InputDockerStreams: []string{"other-main"},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, true)).WithObjects(pipeline).Build()

	usage, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, usage)
}

func TestFindBranchUsage_PipelineAPINotInstalled(t *testing.T) {
	// CDPipeline/Stage kinds are absent from the scheme, emulating a cluster
	// without edp-cd-pipeline-operator: the branch must be reported as unused.
	k8sClient := fake.NewClientBuilder().WithScheme(usageScheme(t, false)).Build()

	usage, err := FindBranchUsage(context.Background(), k8sClient, usageBranch())
	require.NoError(t, err)
	assert.Empty(t, usage)
}
