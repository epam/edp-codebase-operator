package stalecheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestCleanupAction_DeletesUnusedStaleBranch(t *testing.T) {
	branch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)

	scheme := newScheme(t)
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(branch).
		WithStatusSubresource(branch).
		Build()

	recorder := record.NewFakeRecorder(10)
	action := NewCleanupAction(k8sClient, recorder, NewMarkAction(k8sClient, recorder))

	require.NoError(t, action.Apply(context.Background(), branch, Verdict{ExistsInGit: false}))

	err := k8sClient.Get(context.Background(),
		client.ObjectKey{Namespace: testNamespace, Name: "app-feature"}, &codebaseApi.CodebaseBranch{})
	assert.True(t, errors.IsNotFound(err), "stale branch must be deleted")

	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, EventReasonStaleBranchDeleted)
	default:
		t.Fatal("expected StaleBranchDeleted event")
	}
}

func TestCleanupAction_RetainsBranchUsedByCDPipeline(t *testing.T) {
	branch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)

	pipeline := &pipelineApi.CDPipeline{}
	pipeline.Name = "demo"
	pipeline.Namespace = testNamespace
	pipeline.Spec.InputDockerStreams = []string{"app-feature"}
	pipeline.Spec.DeploymentType = "container"

	scheme := newScheme(t)
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(branch, pipeline).
		WithStatusSubresource(branch).
		Build()

	recorder := record.NewFakeRecorder(10)
	action := NewCleanupAction(k8sClient, recorder, NewMarkAction(k8sClient, recorder))

	require.NoError(t, action.Apply(context.Background(), branch, Verdict{ExistsInGit: false}))

	retained := getBranch(t, k8sClient, "app-feature")
	assert.True(t, meta.IsStatusConditionTrue(retained.Status.Conditions, codebaseApi.ConditionStale))
	assert.Equal(t, "true", retained.Labels[codebaseApi.StaleLabel])

	condition := meta.FindStatusCondition(retained.Status.Conditions, codebaseApi.ConditionStale)
	require.NotNil(t, condition)
	assert.Contains(t, condition.Message, "retained because it is used by CDPipeline demo")

	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, EventReasonStaleBranchRetained)
	default:
		t.Fatal("expected StaleBranchRetained event")
	}
}

func TestCleanupAction_ClearsMarkWhenBranchExists(t *testing.T) {
	branch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	branch.Labels = map[string]string{codebaseApi.StaleLabel: "true"}

	scheme := newScheme(t)
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(branch).
		WithStatusSubresource(branch).
		Build()

	recorder := record.NewFakeRecorder(10)
	action := NewCleanupAction(k8sClient, recorder, NewMarkAction(k8sClient, recorder))

	require.NoError(t, action.Apply(context.Background(), branch, Verdict{ExistsInGit: true}))

	updated := getBranch(t, k8sClient, "app-feature")
	assert.NotContains(t, updated.Labels, codebaseApi.StaleLabel)
	assert.False(t, meta.IsStatusConditionTrue(updated.Status.Conditions, codebaseApi.ConditionStale))
}
