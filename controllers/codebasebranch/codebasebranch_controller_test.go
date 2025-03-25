package codebasebranch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestReconcileCodebaseBranch_Reconcile_ShouldPassNotFoundCR(t *testing.T) {
	c := &codebaseApi.CodebaseBranch{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.Codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailGetCodebase(t *testing.T) {
	t.Setenv("WORKING_DIR", "/tmp/1")

	c := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			Pipelines: map[string]string{
				"review": "review-pipeline",
				"build":  "build-pipeline",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	require.Error(t, err)
	assert.False(t, res.Requeue)
	assert.Contains(t, err.Error(), "failed to get Codebase")
}

func TestReconcileCodebaseBranch_Reconcile_ShouldPassDeleteCodebasebranch(t *testing.T) {
	t.Setenv("WORKING_DIR", "/tmp/1")

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
			Finalizers: []string{codebaseBranchOperatorFinalizerName},
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			Pipelines: map[string]string{
				"review": "review-pipeline",
				"build":  "build-pipeline",
			},
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			CiTool: util.CITekton,
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldPassWithCreatingCIS(t *testing.T) {
	t.Setenv("WORKING_DIR", "/tmp/1")

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			BranchName:   "master",
			Pipelines: map[string]string{
				"review": "review-pipeline",
				"build":  "build-pipeline",
			},
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Git: codebaseApi.CodebaseBranchGitStatusBranchCreated,
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}
	config := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: "namespace",
		},
		Data: map[string]string{
			platform.KrciConfigContainerRegistryHost:  "stub-url",
			platform.KrciConfigContainerRegistrySpace: "stub-space",
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-admin-token",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("j-user"),
			"password": []byte("j-token"),
		},
	}
	cis := &codebaseApi.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, coreV1.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(c, cb, s, cis, config).
		WithStatusSubresource(cb).
		Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	cResp := &codebaseApi.CodebaseImageStream{}

	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCodebase-master",
			Namespace: "namespace",
		},
		cResp)
	assert.NoError(t, err)
	assert.Equal(t, "stub-url/stub-space/NewCodebase", cResp.Spec.ImageName)

	gotCodebaseBranch := &codebaseApi.CodebaseBranch{}

	err = fakeCl.Get(context.TODO(), types.NamespacedName{
		Name:      "NewCodebaseBranch",
		Namespace: "namespace",
	}, gotCodebaseBranch)
	if err != nil {
		t.Fatal(err)
	}

	expectedLabels := map[string]string{
		codebasebranch.LabelCodebaseName: "NewCodebase",
	}
	assert.Equal(t, expectedLabels, gotCodebaseBranch.GetLabels())
}

func TestReconcileCodebaseBranch_Reconcile_ShouldRequeueWithCodebaseNotReady(t *testing.T) {
	t.Setenv("WORKING_DIR", "/tmp/1")

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			Pipelines: map[string]string{
				"review": "review-pipeline",
				"build":  "build-pipeline",
			},
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Status: "done",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(c, cb).
		WithStatusSubresource(cb).
		Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.Equal(t, res.RequeueAfter, 5*time.Second)

	gotCodebaseBranch := &codebaseApi.CodebaseBranch{}

	err = fakeCl.Get(context.TODO(), types.NamespacedName{
		Name:      "NewCodebaseBranch",
		Namespace: "namespace",
	}, gotCodebaseBranch)
	if err != nil {
		t.Fatal(err)
	}

	expectedLabels := map[string]string{
		codebasebranch.LabelCodebaseName: "NewCodebase",
	}
	assert.Equal(t, expectedLabels, gotCodebaseBranch.GetLabels())
}

func TestReconcileCodebaseBranch_Reconcile_ShouldInitBuildForEDPVersioning(t *testing.T) {
	t.Parallel()

	namespace := "test-namespace"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			Pipelines: map[string]string{
				"review": "review-pipeline",
				"build":  "build-pipeline",
			},
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: codebaseApi.VersioningTypeSemver,
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(c, cb).
		WithStatusSubresource(cb).
		Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: namespace,
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, res.Requeue)

	gotCodebaseBranch := codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.Background(), types.NamespacedName{
		Name:      "NewCodebaseBranch",
		Namespace: namespace,
	}, &gotCodebaseBranch)

	require.NoError(t, err)

	expectedBuildNumber := "0"

	assert.NotNil(t, gotCodebaseBranch.Status.Build)
	assert.Equal(t, &expectedBuildNumber, gotCodebaseBranch.Status.Build)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldHaveFailStatus(t *testing.T) {
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			BranchName:   "master",
			Pipelines: map[string]string{
				"review": "review-pipeline",
				"build":  "build-pipeline",
			},
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(c, cb).
		WithStatusSubresource(cb).
		Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)

	br := &codebaseApi.CodebaseBranch{}

	err = fakeCl.Get(context.Background(),
		types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		br)
	assert.NoError(t, err)
	assert.Equal(t, codebaseApi.Error, br.Status.Result)
	assert.Contains(t, br.Status.DetailedMessage, "not found")
}

func TestReconcileCodebaseBranch_Reconcile_ShouldSetPipelines(t *testing.T) {
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-branch",
			Namespace: "default",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "test-codebase",
			BranchName:   "test-branch",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-codebase",
			Namespace: "default",
		},
		Spec: codebaseApi.CodebaseSpec{
			Lang:      "go",
			Framework: "gin",
			BuildTool: "go",
			Type:      "application",
			GitServer: "test-gs",
			Versioning: codebaseApi.Versioning{
				Type: codebaseApi.VersioningTypDefault,
			},
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-gs",
			Namespace: "default",
		},
		Spec: codebaseApi.GitServerSpec{
			GitProvider: codebaseApi.GitProviderGithub,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(c, cb, gs).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-branch",
			Namespace: "default",
		},
	}

	controller := NewReconcileCodebaseBranch(fakeCl, scheme, logr.Discard())

	res, err := controller.Reconcile(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, res.Requeue)

	updatedCb := &codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(
		context.Background(),
		types.NamespacedName{
			Name:      "test-branch",
			Namespace: "default",
		},
		updatedCb,
	)

	require.NoError(t, err)
	require.Len(t, updatedCb.Spec.Pipelines, 2)
	require.Contains(t, updatedCb.Spec.Pipelines, "review")
	assert.Equal(t, "github-go-gin-app-review", updatedCb.Spec.Pipelines["review"])
	require.Contains(t, updatedCb.Spec.Pipelines, "build")
	assert.Equal(t, "github-go-gin-app-build-default", updatedCb.Spec.Pipelines["build"])
}

func TestReconcileCodebaseBranch_Reconcile_FailedToSetPipelines_GitServerNotFound(t *testing.T) {
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-branch",
			Namespace: "default",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "test-codebase",
			BranchName:   "test-branch",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-codebase",
			Namespace: "default",
		},
		Spec: codebaseApi.CodebaseSpec{
			Lang:      "go",
			Framework: "gin",
			BuildTool: "go",
			Type:      "application",
			GitServer: "test-gs",
			Versioning: codebaseApi.Versioning{
				Type: codebaseApi.VersioningTypDefault,
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(c, cb).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-branch",
			Namespace: "default",
		},
	}

	controller := NewReconcileCodebaseBranch(fakeCl, scheme, logr.Discard())

	_, err := controller.Reconcile(context.Background(), req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get GitServer")
}
