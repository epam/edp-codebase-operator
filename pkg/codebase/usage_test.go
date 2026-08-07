package codebase

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

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/deploymentusage"
)

func codebaseUsageScheme(t *testing.T, withPipelineAPI bool) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	if withPipelineAPI {
		require.NoError(t, pipelineApi.AddToScheme(scheme))
	}

	return scheme
}

// codebaseUsageClientBuilder returns a fake client builder with BranchCodebaseNameIndex
// registered. A fake client is not a client.FieldIndexer, so it takes the index through
// the builder; a List selecting on an unregistered index fails.
func codebaseUsageClientBuilder(t *testing.T, withPipelineAPI bool) *fake.ClientBuilder {
	t.Helper()

	return fake.NewClientBuilder().
		WithScheme(codebaseUsageScheme(t, withPipelineAPI)).
		WithIndex(&codebaseApi.CodebaseBranch{}, BranchCodebaseNameIndex, IndexBranchByCodebaseName)
}

func usageCodebase() *codebaseApi.Codebase {
	return &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
	}
}

func TestFindCodebaseUsage_Applications(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications: []string{"other-app", "app"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(pipeline).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "CDPipeline demo (applications)", refs[0].String())
}

func TestFindCodebaseUsage_ApplicationsToPromote(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications:          []string{"other-app"},
			ApplicationsToPromote: []string{"app"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(pipeline).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "CDPipeline demo (applicationsToPromote)", refs[0].String())
}

func TestFindCodebaseUsage_ApplicationsTakesPrecedence(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications:          []string{"app"},
			ApplicationsToPromote: []string{"app"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(pipeline).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "CDPipeline demo (applications)", refs[0].String())
}

// Deleting a Codebase cascades to its branches, and the CodebaseBranch webhook lets that
// cascade through, so a pipeline consuming one of those branches must block the Codebase.
func TestFindCodebaseUsage_BranchInInputDockerStreams(t *testing.T) {
	branch := &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{Name: "app-main", Namespace: "default"},
		Spec:       codebaseApi.CodebaseBranchSpec{CodebaseName: "app", BranchName: "main"},
	}
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications:       []string{"other-app"},
			InputDockerStreams: []string{"app-main"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).
		WithObjects(branch, pipeline).
		Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "CDPipeline demo (branch app-main in inputDockerStreams)", refs[0].String())
	assert.Equal(t, deploymentusage.FieldInputDockerStreams, refs[0].Field)
}

func TestFindCodebaseUsage_ForeignBranchInInputDockerStreams(t *testing.T) {
	branch := &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{Name: "other-app-main", Namespace: "default"},
		Spec:       codebaseApi.CodebaseBranchSpec{CodebaseName: "other-app", BranchName: "main"},
	}
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications:       []string{"other-app"},
			InputDockerStreams: []string{"other-app-main"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).
		WithObjects(branch, pipeline).
		Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// A List failure that is not "CRD absent" must surface, so the webhook denies rather than
// silently treating an unverifiable codebase as unused.
func TestFindCodebaseUsage_ListErrorIsSurfaced(t *testing.T) {
	k8sClient := codebaseUsageClientBuilder(t, true).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return apierrors.NewForbidden(
					schema.GroupResource{Group: "v2.edp.epam.com", Resource: "cdpipelines"}, "", errors.New("no access"))
			},
		}).
		Build()

	_, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list")
}

// The branch lookup must stay scoped to the codebase through BranchCodebaseNameIndex.
// The refs are the same whether or not it is scoped, so the selector itself is the only
// thing that can be asserted on.
func TestFindCodebaseUsage_BranchLookupIsIndexed(t *testing.T) {
	var branchListOpts []client.ListOption

	k8sClient := codebaseUsageClientBuilder(t, true).
		WithObjects(&codebaseApi.CodebaseBranch{
			ObjectMeta: metav1.ObjectMeta{Name: "app-main", Namespace: "default"},
			Spec:       codebaseApi.CodebaseBranchSpec{CodebaseName: "app", BranchName: "main"},
		}).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(
				ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption,
			) error {
				if _, ok := list.(*codebaseApi.CodebaseBranchList); ok {
					branchListOpts = opts
				}

				return c.List(ctx, list, opts...)
			},
		}).
		Build()

	_, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	assert.Contains(t, branchListOpts, client.MatchingFields{BranchCodebaseNameIndex: "app"})
}

func TestFindCodebaseUsage_AutotestQualityGate(t *testing.T) {
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("main"),
			}},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(stage).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "Stage demo-dev of CDPipeline demo (autotest quality gate)", refs[0].String())
}

// A manual quality gate leaves AutotestName unset and must never be treated as a match.
func TestFindCodebaseUsage_ManualQualityGateNotMatched(t *testing.T) {
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

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(stage).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindCodebaseUsage_MultipleReferences(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications: []string{"app"},
		},
	}
	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			CdPipeline: "demo",
			QualityGates: []pipelineApi.QualityGate{{
				QualityGateType: "autotests",
				AutotestName:    ptr.To("app"),
				BranchName:      ptr.To("main"),
			}},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(pipeline, stage).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	require.Len(t, refs, 2)

	descriptions := []string{refs[0].String(), refs[1].String()}
	assert.Contains(t, descriptions, "CDPipeline demo (applications)")
	assert.Contains(t, descriptions, "Stage demo-dev of CDPipeline demo (autotest quality gate)")
}

func TestFindCodebaseUsage_TerminatingPipelineIgnored(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "demo",
			Namespace:         "default",
			DeletionTimestamp: ptr.To(metav1.Now()),
			Finalizers:        []string{"keep"},
		},
		Spec: pipelineApi.CDPipelineSpec{
			Applications: []string{"app"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(pipeline).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindCodebaseUsage_Unused(t *testing.T) {
	pipeline := &pipelineApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: pipelineApi.CDPipelineSpec{
			Applications: []string{"other-app"},
		},
	}

	k8sClient := codebaseUsageClientBuilder(t, true).WithObjects(pipeline).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindCodebaseUsage_PipelineAPINotInstalled(t *testing.T) {
	// CDPipeline/Stage kinds are absent from the scheme, emulating a cluster
	// without edp-cd-pipeline-operator: the codebase must be reported as unused.
	k8sClient := codebaseUsageClientBuilder(t, false).Build()

	refs, err := FindCodebaseUsage(context.Background(), k8sClient, usageCodebase())
	require.NoError(t, err)
	assert.Empty(t, refs)
}
