package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestDeleteJiraIssueMetadataCr_ServeRequest(t *testing.T) {
	jim := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
	}

	objs := []runtime.Object{
		jim,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, jim)

	dimcr := DeleteJiraIssueMetadataCr{
		c: fake.NewFakeClient(objs...),
	}

	err := dimcr.ServeRequest(jim)
	assert.NoError(t, err)
}
