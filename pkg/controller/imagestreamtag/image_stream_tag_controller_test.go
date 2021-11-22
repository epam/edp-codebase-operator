package imagestreamtag

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

func TestReconcileImageStreamTag_Reconcile_ShouldPassNotFound(t *testing.T) {
	ist := &v1alpha1.ImageStreamTag{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "IST",
			Namespace: "namespace",
		},
	}

	r := ReconcileImageStreamTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileImageStreamTag_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "IST",
			Namespace: "namespace",
		},
	}

	r := ReconcileImageStreamTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.ImageStreamTag") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileImageStreamTag_Reconcile_ShouldFail(t *testing.T) {
	ist := &v1alpha1.ImageStreamTag{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "IST",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ImageStreamTagSpec{
			Tag:                     "111",
			CodebaseImageStreamName: "cis",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "IST",
			Namespace: "namespace",
		},
	}

	r := ReconcileImageStreamTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "couldn't add tag to codebase image stream cis") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileImageStreamTag_Reconcile_ShouldPass(t *testing.T) {
	ist := &v1alpha1.ImageStreamTag{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "IST",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ImageStreamTagSpec{
			Tag:                     "111",
			CodebaseImageStreamName: "codebase-master",
		},
	}
	cis := &v1alpha1.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase-master",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, cis).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "IST",
			Namespace: "namespace",
		},
	}

	r := ReconcileImageStreamTag{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	istResp := &v1alpha1.CodebaseImageStream{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "codebase-master",
			Namespace: "namespace",
		},
		istResp)
	assert.NoError(t, err)
	assert.Equal(t, len(istResp.Spec.Tags), 1)
	assert.Equal(t, istResp.Spec.Tags[0].Name, "111")
}
