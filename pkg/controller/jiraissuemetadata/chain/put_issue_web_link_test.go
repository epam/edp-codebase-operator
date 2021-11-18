package chain

import (
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPutIssueWebLink_ServeRequest_ShouldPass(t *testing.T) {
	mClient := new(mock.MockClient)
	mClient.On("CreateIssueLink", "fake-issueId", "fake-title", "fake-url").Return(
		nil)

	jim := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			Payload: `{"issuesLinks": [{"ticket":"fake-issueId", "title":"fake-title", "url":"fake-url"}]}`,
		},
	}

	piwl := PutIssueWebLink{
		client: mClient,
	}

	err := piwl.ServeRequest(jim)
	assert.NoError(t, err)
}

func TestPutIssueWebLink_ServeRequest_ShouldFail(t *testing.T) {
	mClient := new(mock.MockClient)
	mClient.On("CreateIssueLink", "fake-issueId", "fake-title", "fake-url").Return(
		nil)

	jim := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			Payload: "{}",
		},
	}

	piwl := PutIssueWebLink{
		client: mClient,
	}

	err := piwl.ServeRequest(jim)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "issuesLinks is a mandatory field in payload") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
