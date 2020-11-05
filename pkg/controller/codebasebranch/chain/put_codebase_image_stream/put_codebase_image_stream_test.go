package put_codebase_image_stream

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpV1alpha1 "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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

	objs := []runtime.Object{
		c, ec, cis,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, ec)
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}, cis)
	client := fake.NewFakeClient(objs...)
	cisChain := PutCodebaseImageStream{
		Client: client,
	}

	err := cisChain.ServeRequest(cb)
	assert.NoError(t, err)

	cisResp := &v1alpha1.CodebaseImageStream{}
	err = client.Get(nil,
		types.NamespacedName{
			Name:      "stub-name-stub-name",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
}

func TestPutCodebaseImageStream_ShouldCreateCisWithEdpVersioningType(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "edp",
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

	objs := []runtime.Object{
		c, ec, cis,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, ec)
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}, cis)
	client := fake.NewFakeClient(objs...)
	cisChain := PutCodebaseImageStream{
		Client: client,
	}

	err := cisChain.ServeRequest(cb)
	assert.NoError(t, err)

	cisResp := &v1alpha1.CodebaseImageStream{}
	err = client.Get(nil,
		types.NamespacedName{
			Name:      "stub-name-edp-stub-name",
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

	objs := []runtime.Object{
		c,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c)
	client := fake.NewFakeClient(objs...)
	cisChain := PutCodebaseImageStream{
		Client: client,
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

	objs := []runtime.Object{
		c, ec,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, ec)
	client := fake.NewFakeClient(objs...)
	cisChain := PutCodebaseImageStream{
		Client: client,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)
}
