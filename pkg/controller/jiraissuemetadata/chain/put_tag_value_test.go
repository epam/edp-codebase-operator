package chain

import (
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mock"
)

func TestPutTagValue_ServeRequest(t *testing.T) {
	jiraProject := &jira.Project{
		Key: "fake-projectKey",
		ID:  "fake-projectId",
	}

	issueMetadata := &jira.CreateMetaInfo{
		Projects: []*jira.MetaProject{
			{
				Id: "fake-projectId",
				IssueTypes: []*jira.MetaIssueType{
					{
						Name: "fake-type",
					},
				},
			},
		},
	}

	issueType := "fake-type"

	mClient := new(mock.MockClient)
	mClient.On("CreateFixVersionValue", "fake-projectId", "fake-versionName").Return(
		nil)
	mClient.On("CreateComponentValue", "fake-projectId", "fake-componentName").Return(
		nil)
	mClient.On("GetProjectInfo", "fake-issueId").Return(
		jiraProject, nil)
	mClient.On("GetIssueType", "fake-issueId").Return(
		&issueType, nil)
	mClient.On("GetIssueMetadata", "fake-projectKey").Return(
		issueMetadata, nil)

	jim := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			Tickets: []string{"fake-issueId"},
			Payload: `{"issuesLinks": [{"ticket":"fake-issueId", "title":"fake-title", "url":"fake-url"}], "allowedValues": [{"ticket":"fakeId"}]}`,
		},
	}

	ptv := PutTagValue{
		client: mClient,
	}

	err := ptv.ServeRequest(jim)
	assert.NoError(t, err)
}
