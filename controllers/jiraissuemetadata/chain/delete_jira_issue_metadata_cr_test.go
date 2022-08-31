package chain

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestDeleteJiraIssueMetadataCr_ServeRequest(t *testing.T) {
	jim := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
	}

	scheme.Scheme.AddKnownTypes(metaV1.SchemeGroupVersion, jim)

	newScheme := runtime.NewScheme()
	newScheme.AddKnownTypes(codebaseApi.GroupVersion, jim)
	fakeCl := fake.NewClientBuilder().WithScheme(newScheme).WithRuntimeObjects(jim).Build()

	dimcr := DeleteJiraIssueMetadataCr{
		c: fakeCl,
	}

	err := dimcr.ServeRequest(jim)
	assert.NoError(t, err)
}

func TestDeleteJiraIssueMetadataCr_ServeRequest_StopOnErrors(t *testing.T) {
	jim := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Status: codebaseApi.JiraIssueMetadataStatus{
			Error: multierror.Append(errors.New("error1"), errors.New("error2")),
		},
	}

	scheme.Scheme.AddKnownTypes(metaV1.SchemeGroupVersion, jim)

	sch1 := runtime.NewScheme()
	sch1.AddKnownTypes(codebaseApi.GroupVersion, jim)
	fakeCl := fake.NewClientBuilder().WithScheme(sch1).WithRuntimeObjects(jim).Build()

	dimcr := DeleteJiraIssueMetadataCr{
		c: fakeCl,
	}

	err := dimcr.ServeRequest(jim)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "error1\nerror2")
}
