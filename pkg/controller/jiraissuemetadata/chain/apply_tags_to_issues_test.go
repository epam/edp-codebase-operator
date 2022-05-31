package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mock"
)

func TestApplyTagsToIssues_ServeRequest(t *testing.T) {
	mClient := new(mock.MockClient)
	mClient.On("ApplyTagsToIssue", "fake-issue", `{"tags": "fake-tags"}`).Return(
		nil)

	jim := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			Payload: `{"issuesLinks": [{"ticket":"fake-issueId", "title":"fake-title", "url":"fake-url"}]}`,
		},
	}

	atti := ApplyTagsToIssues{
		client: mClient,
	}

	err := atti.ServeRequest(jim)
	assert.NoError(t, err)
}
