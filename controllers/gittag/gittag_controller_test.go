package gittag

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestReconcileGitTag_Reconcile_ShouldPassNotFound(t *testing.T) {
	gt := &codebaseApi.GitTag{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gt)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gt).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGT",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitTag{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileGitTag_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGT",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitTag{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.GitTag") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}

func TestReconcileGitTag_Reconcile_ShouldFail(t *testing.T) {
	gt := &codebaseApi.GitTag{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewGT",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitTagSpec{
			Tag: "111",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gt)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gt).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewGT",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitTag{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "couldn't push add tag 111") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}
