package gittag

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileGitTag_SetupWithManager(t *testing.T) {
	r := NewReconcileGitTag(nil, logr.DiscardLogger{})
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{MetricsBindAddress: "0"})
	if err != nil {
		t.Fatal(err)
	}

	err = r.SetupWithManager(mgr)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "no kind is registered for the type") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileGitTag_Reconcile_ShouldPassNotFound(t *testing.T) {
	gt := &v1alpha1.GitTag{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gt)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gt).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGT",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileGitTag_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGT",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.GitTag") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileGitTag_Reconcile_ShouldFail(t *testing.T) {
	gt := &v1alpha1.GitTag{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewGT",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GitTagSpec{
			Tag: "111",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gt)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gt).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGT",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "couldn't push add tag 111") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}
