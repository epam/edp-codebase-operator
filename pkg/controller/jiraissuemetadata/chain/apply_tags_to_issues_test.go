package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestApplyTagsToIssues_ServeRequest(t *testing.T) {
	mClient := new(mock.MockClient)
	mClient.On("ApplyTagsToIssue", "fake-issue", `{"tags": "fake-tags"}`).Return(
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

	atti := ApplyTagsToIssues{
		client: mClient,
	}

	err := atti.ServeRequest(jim)
	assert.NoError(t, err)
}
