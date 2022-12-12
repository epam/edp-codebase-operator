package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutTagCodebaseImageStreamCrChain_ShouldBeExecutedSuccessfullyIfTagExistsInCIS(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Tags: []codebaseApi.Tag{
				{
					Name: "fake-tag",
				},
				{
					Name: "fake-tag2",
				},
			},
		},
	}

	objs := []runtime.Object{cis}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cis)

	ch := PutTagCodebaseImageStreamCr{
		client: fake.NewClientBuilder().WithRuntimeObjects(objs...).Build(),
	}

	ist := &codebaseApi.ImageStreamTag{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "fake-namespace",
		},
		Spec: codebaseApi.ImageStreamTagSpec{
			CodebaseImageStreamName: "fake-name",
			Tag:                     "fake-tag",
		},
	}
	err := ch.ServeRequest(ist)
	assert.NoError(t, err)
}

func TestPutTagCodebaseImageStreamCrChain_ShouldBeExecutedSuccessfullyIfTagDoesntExistInCIS(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
	}

	objs := []runtime.Object{cis}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cis)

	ch := PutTagCodebaseImageStreamCr{
		client: fake.NewClientBuilder().WithRuntimeObjects(objs...).Build(),
	}

	ist := &codebaseApi.ImageStreamTag{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "fake-namespace",
		},
		Spec: codebaseApi.ImageStreamTagSpec{
			CodebaseImageStreamName: "fake-name",
			Tag:                     "fake-tag",
		},
	}
	err := ch.ServeRequest(ist)
	assert.NoError(t, err)
}
