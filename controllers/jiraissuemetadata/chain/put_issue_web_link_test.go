package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	jiraMock "github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mocks"
)

func TestPutIssueWebLink_ServeRequest_ShouldPass(t *testing.T) {
	mClient := jiraMock.NewMockClient(t)
	mClient.On("CreateIssueLink", "fake-issueId", "fake-title", "fake-url").Return(
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

	piwl := PutIssueWebLink{
		client: mClient,
	}

	err := piwl.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), jim)
	assert.NoError(t, err)
}

func TestPutIssueWebLink_ServeRequest_ShouldFail(t *testing.T) {
	mClient := jiraMock.NewMockClient(t)
	mClient.On("CreateIssueLink", "DEV-0000",
		"[DEV-0000] updated components versions [alpha-zeta][build/1.5.0-SNAPSHOT.377]",
		"https://example.com/job/alpha-zeta/job/MASTER-Build-alpha-zeta/890/console").Return(
		errors.New("create-link-failure"))

	jim := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			Payload: "{\n        \"components\": \"control-plane-gerrit\",\n        \"issuesLinks\": [\n" +
				"            {\n" +
				"                \"ticket\": \"DEV-0000\",\n" +
				"                \"title\": \"[DEV-0000] updated components versions [alpha-zeta][build/1.5.0-SNAPSHOT.377]\",\n" +
				"                \"url\": \"https://example.com/job/alpha-zeta/job/MASTER-Build-alpha-zeta/890/console\"\n" +
				"            }\n" +
				"        ],\n" +
				"        \"fixVersions\": \"alpha-zeta-1.5.0\"\n" +
				"    }",
		},
	}

	piwl := PutIssueWebLink{
		client: mClient,
	}

	err := piwl.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), jim)
	assert.NoError(t, err)
	assert.NotNil(t, jim.Status.ErrorStrings)
}
