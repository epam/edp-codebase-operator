package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestUpdateGitStatusWithPatch_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: "",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)

	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)

	// Verify status was actually patched in the cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, updated.Status.Git)
}

func TestUpdateGitStatusWithPatch_Idempotent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: util.ProjectPushedStatus,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)

	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)
}

func TestUpdateGitStatusWithPatch_StatusTransition(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: util.ProjectPushedStatus,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	// Execute - transition from ProjectPushedStatus to ProjectGitLabCIPushedStatus
	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectGitLabCIPushedStatus,
	)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)

	// Verify in cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectGitLabCIPushedStatus, updated.Status.Git)
}

func TestUpdateGitStatusWithPatch_PreservesOtherStatusFields(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git:        "",
			WebHookRef: "webhook-123",
			GitWebUrl:  "https://git.example.com/repo",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	// Execute
	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)

	// Verify - only Git field changed, others preserved
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)
	assert.Equal(t, "webhook-123", codebase.Status.WebHookRef)
	assert.Equal(t, "https://git.example.com/repo", codebase.Status.GitWebUrl)

	// Verify in cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, updated.Status.Git)
	assert.Equal(t, "webhook-123", updated.Status.WebHookRef)
	assert.Equal(t, "https://git.example.com/repo", updated.Status.GitWebUrl)
}

func TestUpdateGitStatusWithPatch_SequentialUpdates(t *testing.T) {
	// Setup - This test simulates the chain handler scenario
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: "",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	ctx := context.Background()

	// Simulate sequential updates as in chain handlers
	// 1. PutProject sets ProjectPushedStatus
	err := updateGitStatusWithPatch(
		ctx,
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)

	// 2. PutGitLabCIConfig sets ProjectGitLabCIPushedStatus
	err = updateGitStatusWithPatch(
		ctx,
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectGitLabCIPushedStatus,
	)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)

	// 3. PutDeployConfigs sets ProjectTemplatesPushedStatus
	err = updateGitStatusWithPatch(
		ctx,
		fakeClient,
		codebase,
		codebaseApi.SetupDeploymentTemplates,
		util.ProjectTemplatesPushedStatus,
	)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectTemplatesPushedStatus, codebase.Status.Git)

	// Verify final state in cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectTemplatesPushedStatus, updated.Status.Git)
}
