package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestPutTagCodebaseImageStreamCrChain_ShouldBeExecutedSuccessfullyIfTagExistsInCIS(t *testing.T) {
	cis := &v1alpha1.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.CodebaseImageStreamSpec{
			Tags: []v1alpha1.Tag{
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
		client: fake.NewFakeClient(objs...),
	}

	ist := &v1alpha1.ImageStreamTag{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.ImageStreamTagSpec{
			CodebaseImageStreamName: "fake-name",
			Tag:                     "fake-tag",
		},
	}
	err := ch.ServeRequest(ist)
	assert.NoError(t, err)
}

func TestPutTagCodebaseImageStreamCrChain_ShouldBeExecutedSuccessfullyIfTagDoesntExistInCIS(t *testing.T) {
	cis := &v1alpha1.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
	}

	objs := []runtime.Object{cis}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cis)

	ch := PutTagCodebaseImageStreamCr{
		client: fake.NewFakeClient(objs...),
	}

	ist := &v1alpha1.ImageStreamTag{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.ImageStreamTagSpec{
			CodebaseImageStreamName: "fake-name",
			Tag:                     "fake-tag",
		},
	}
	err := ch.ServeRequest(ist)
	assert.NoError(t, err)
}
