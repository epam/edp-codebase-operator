package gitserver

import (
	"context"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileGitServer_Reconcile_ShouldPassNotFound(t *testing.T) {
	gs := &v1alpha1.GitServer{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGitServer",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileGitServer_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGitServer",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.GitServer") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileGitServer_Reconcile_ShouldFailToGetSecret(t *testing.T) {
	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewGitServer",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGitServer",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "an error has occurred  while getting ssh-secret secret") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileGitServer_UpdateStatus_ShouldPassWithSuccess(t *testing.T) {
	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewGitServer",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	r := ReconcileGitServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	err := r.updateStatus(context.TODO(), fakeCl, gs, true)

	assert.NoError(t, err)
	assert.Equal(t, gs.Status.Result, "success")
}

func TestReconcileGitServer_UpdateStatus_ShouldPassWithFailure(t *testing.T) {
	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewGitServer",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	r := ReconcileGitServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	err := r.updateStatus(context.TODO(), fakeCl, gs, false)

	assert.NoError(t, err)
	assert.Equal(t, gs.Status.Result, "error")
}
