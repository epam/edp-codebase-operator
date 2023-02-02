package put_codebase_image_stream

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	edpComponentApi "github.com/epam/edp-component-operator/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutCodebaseImageStream_ShouldCreateCisWithDefaultVersioningType(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "default",
			},
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name",
		},
	}

	ec := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerRegistryName,
			Namespace: "stub-namespace",
		},
		Spec: edpComponentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, ec, cb)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, ec, cis).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.NoError(t, err)

	cisResp := &codebaseApi.CodebaseImageStream{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name-stub-name",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
}

func TestPutCodebaseImageStream_ShouldNotFindCodebase(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "default",
			},
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name-fake",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)
}

func TestPutCodebaseImageStream_ShouldNotFindEdpComponent(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "default",
			},
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name",
		},
	}

	ec := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "stub-namespace",
		},
		Spec: edpComponentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, ec)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)
}

func TestPutCodebaseImageStream_ShouldFailToGetCodebase(t *testing.T) {
	c := &codebaseApi.Codebase{}
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "stub-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()
	cisChain := PutCodebaseImageStream{
		Client: fakeCl,
	}

	err := cisChain.ServeRequest(cb)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to get Codebase stub-name") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
