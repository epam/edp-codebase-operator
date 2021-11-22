package codebaseimagestream

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

func TestReconcileCodebaseImageStream_Reconcile_ShouldPassNotFound(t *testing.T) {
	gs := &v1alpha1.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseImageStream_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.CodebaseImageStream") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseImageStream_Reconcile_ShouldPass(t *testing.T) {
	gs := &v1alpha1.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseImageStream_Reconcile_ShouldFail(t *testing.T) {
	gs := &v1alpha1.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCIS",
			Namespace: "namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "cb-name",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "couldn't handle NewCIS codebase image stream") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
