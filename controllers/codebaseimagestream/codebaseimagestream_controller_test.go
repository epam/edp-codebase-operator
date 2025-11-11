package codebaseimagestream

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestReconcileCodebaseImageStream_Reconcile_ShouldPassNotFound(t *testing.T) {
	gs := &codebaseApi.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)
}

func TestReconcileCodebaseImageStream_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.CodebaseImageStream") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Equal(t, time.Duration(0), res.RequeueAfter)
}

func TestReconcileCodebaseImageStream_Reconcile_ShouldPass(t *testing.T) {
	gs := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)
}

func TestReconcileCodebaseImageStream_Reconcile_ShouldFail(t *testing.T) {
	gs := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCIS",
			Namespace: "namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "cb-name",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCIS",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseImageStream{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)

	if !strings.Contains(err.Error(), "failed to handle NewCIS codebase image stream") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
