package chain

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-multierror"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteJiraIssueMetadataCr_ServeRequest(t *testing.T) {
	jim := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
	}

	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, jim)
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, jim)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(jim).Build()

	dimcr := DeleteJiraIssueMetadataCr{
		c: fakeCl,
	}

	err := dimcr.ServeRequest(jim)
	assert.NoError(t, err)
}

func TestDeleteJiraIssueMetadataCr_ServeRequest_StopOnErrors(t *testing.T) {

	jim := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Status: v1alpha1.JiraIssueMetadataStatus{
			Error: multierror.Append(errors.New("error1"), errors.New("error2")),
		},
	}

	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, jim)
	sch1 := runtime.NewScheme()
	sch1.AddKnownTypes(v1alpha1.SchemeGroupVersion, jim)
	fakeCl := fake.NewClientBuilder().WithScheme(sch1).WithRuntimeObjects(jim).Build()

	dimcr := DeleteJiraIssueMetadataCr{
		c: fakeCl,
	}

	err := dimcr.ServeRequest(jim)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "error1\nerror2")
}
