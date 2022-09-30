package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
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
	newScheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, jim)
	fakeCl := fake.NewClientBuilder().WithScheme(newScheme).WithRuntimeObjects(jim).Build()

	dimcr := DeleteJiraIssueMetadataCr{
		c: fakeCl,
	}

	err := dimcr.ServeRequest(jim)
	assert.NoError(t, err)
}
