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

	edpComponentApi "github.com/epam/edp-component-operator/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
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

	assert.Error(t, err)
	assert.False(t, res.Requeue)

	if !strings.Contains(err.Error(), "failed to get Codebase ") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailDeleteCodebasebranch(t *testing.T) {
	t.Setenv("WORKING_DIR", "/tmp/1")

	c := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	cb := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
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

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)

	if !strings.Contains(err.Error(), "failed to remove codebasebranch NewCodebaseBranch") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
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
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
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

	ec := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "docker-registry",
			Namespace: "namespace",
		},
		Spec: edpComponentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}
	cis := &codebaseApi.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, edpComponentApi.AddToScheme(scheme))
	require.NoError(t, coreV1.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, s, ec, cis).Build()

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
	assert.Equal(t, cResp.Spec.ImageName, "stub-url/NewCodebase")

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
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: codebaseApi.VersioningTypeEDP,
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

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
