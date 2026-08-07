package deploymentusage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

// The rendered form reaches clients verbatim: it is the message of the admission denial
// and of each StatusCause.
func TestReference_String(t *testing.T) {
	tests := []struct {
		name string
		ref  Reference
		want string
	}{
		{
			name: "CDPipeline reference",
			ref: Reference{
				Kind:   KindCDPipeline,
				Name:   "demo",
				Field:  "spec.inputDockerStreams",
				Reason: "inputDockerStreams",
			},
			want: "CDPipeline demo (inputDockerStreams)",
		},
		{
			name: "Stage reference names its parent CDPipeline",
			ref: Reference{
				Kind:             KindStage,
				Name:             "demo-dev",
				Field:            "spec.qualityGates.autotestName",
				Reason:           "autotest quality gate",
				ParentCDPipeline: "demo",
			},
			want: "Stage demo-dev of CDPipeline demo (autotest quality gate)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ref.String())
		})
	}
}

func TestJoin(t *testing.T) {
	cdPipeline := Reference{Kind: KindCDPipeline, Name: "demo", Reason: "applications"}
	stage := Reference{Kind: KindStage, Name: "demo-dev", Reason: "autotest quality gate", ParentCDPipeline: "demo"}

	tests := []struct {
		name string
		refs []Reference
		want string
	}{
		{
			name: "no references",
			refs: nil,
			want: "",
		},
		{
			name: "single reference is rendered without a separator",
			refs: []Reference{cdPipeline},
			want: "CDPipeline demo (applications)",
		},
		{
			name: "multiple references are separated by a semicolon",
			refs: []Reference{cdPipeline, stage},
			want: "CDPipeline demo (applications); Stage demo-dev of CDPipeline demo (autotest quality gate)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Join(tt.refs))
		})
	}
}

func gateScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	return scheme
}

// A Stage on its way out no longer deploys anything, so its gates must not keep a
// Codebase or CodebaseBranch alive.
func TestFindAutotestGates_TerminatingStageIgnored(t *testing.T) {
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "demo-dev",
			Namespace:         "default",
			DeletionTimestamp: ptr.To(metav1.Now()),
			Finalizers:        []string{"keep"},
		},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("main"),
			}},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(gateScheme(t)).WithObjects(stage).Build()

	gates, err := FindAutotestGates(context.Background(), k8sClient, "default")
	require.NoError(t, err)
	assert.Empty(t, gates)
}

// A gate that names no autotest carries no reference, and must not reach callers as one
// with an empty AutotestName.
func TestFindAutotestGates_SkipsGatesWithoutAutotest(t *testing.T) {
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{
				{QualityGateType: "manual", StepName: "approve"},
				{QualityGateType: "autotests", AutotestName: ptr.To("")},
				{QualityGateType: "autotests", AutotestName: ptr.To("app")},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(gateScheme(t)).WithObjects(stage).Build()

	gates, err := FindAutotestGates(context.Background(), k8sClient, "default")
	require.NoError(t, err)
	require.Len(t, gates, 1)
	assert.Equal(t, "app", gates[0].AutotestName)
	// An unset BranchName must not be reported as a branch of its own.
	assert.Empty(t, gates[0].BranchName)
}

// A Stage listing failure means usage could not be verified and must surface.
func TestFindAutotestGates_ListErrorIsSurfaced(t *testing.T) {
	k8sClient := fake.NewClientBuilder().
		WithScheme(gateScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return apierrors.NewForbidden(
					schema.GroupResource{Group: "v2.edp.epam.com", Resource: "stages"}, "", errors.New("no access"))
			},
		}).
		Build()

	_, err := FindAutotestGates(context.Background(), k8sClient, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list Stages")
}
