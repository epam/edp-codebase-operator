package put_codebase_image_stream

import (
	"context"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpV1alpha1 "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutCodebaseImageStream_ShouldCreateCisWithDefaultVersioningType(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "default",
			},
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name",
		},
	}

	ec := &edpV1alpha1.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dockerRegistryName,
			Namespace: "stub-namespace",
		},
		Spec: edpV1alpha1.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	cis := &v1alpha1.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, ec, cb)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, ec, cis).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.NoError(t, err)

	cisResp := &v1alpha1.CodebaseImageStream{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name-stub-name",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
}

func TestPutCodebaseImageStream_ShouldNotFindCodebase(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "default",
			},
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name-fake",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)
}

func TestPutCodebaseImageStream_ShouldNotFindEdpComponent(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "default",
			},
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name",
		},
	}

	ec := &edpV1alpha1.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "stub-namespace",
		},
		Spec: edpV1alpha1.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, ec)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)
}

func TestPutCodebaseImageStream_ShouldFailToGetCodebase(t *testing.T) {
	c := &v1alpha1.Codebase{}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "Unable to get Codebase stub-name") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
