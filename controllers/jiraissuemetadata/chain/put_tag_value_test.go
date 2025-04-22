package chain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	goJira "github.com/andygrunwald/go-jira"
	"github.com/go-logr/logr"
	testify "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	jiraMock "github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mocks"
)

func TestPutTagValue_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata *codebaseApi.JiraIssueMetadata
		client   func(t *testing.T) jira.Client
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "successfully put tag value",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "1", "2").
					Return(map[string]jira.IssueTypeMeta{
						"fixVersions": {
							FieldID: "fixVersions",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "0.0.2-snapshot-java11",
							}},
						},
						"components": {
							FieldID: "components",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "java11",
							}},
						},
					}, nil)
				m.On("CreateFixVersionValue", testify.Anything, 1, "0.0.1-snapshot-java11").
					Return(nil)
				m.On("CreateComponentValue", testify.Anything, 1, "java11-mvn-create").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "tag values already exist",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "1", "2").
					Return(map[string]jira.IssueTypeMeta{
						"fixVersions": {
							FieldID: "fixVersions",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "0.0.1-snapshot-java11",
							}},
						},
						"components": {
							FieldID: "components",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "java11-mvn-create",
							}},
						},
					}, nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "issue type meta doesn't contain fix version field",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "1", "2").
					Return(map[string]jira.IssueTypeMeta{
						"components": {
							FieldID: "components",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "java11",
							}},
						},
					}, nil)
				m.On("CreateComponentValue", testify.Anything, 1, "java11-mvn-create").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to put fix version value",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "1", "2").
					Return(map[string]jira.IssueTypeMeta{
						"fixVersions": {
							FieldID: "fixVersions",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "0.0.2-snapshot-java11",
							}},
						},
						"components": {
							FieldID: "components",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "java11",
							}},
						},
					}, nil)
				m.On("CreateFixVersionValue", testify.Anything, 1, "0.0.1-snapshot-java11").
					Return(errors.New("failed"))
				m.On("CreateComponentValue", testify.Anything, 1, "java11-mvn-create").Maybe().
					Return(nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create value")
			},
		},
		{
			name: "invalid project ID",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "abc",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "abc", "2").
					Return(map[string]jira.IssueTypeMeta{
						"fixVersions": {
							FieldID: "fixVersions",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "0.0.2-snapshot-java11",
							}},
						},
						"components": {
							FieldID: "components",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "java11",
							}},
						},
					}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to parse to int project ID")
			},
		},
		{
			name: "invalid payload fix version value",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":{\"data\":\"0.0.1-snapshot-java11\"},\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "1", "2").
					Return(map[string]jira.IssueTypeMeta{
						"fixVersions": {
							FieldID: "fixVersions",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "0.0.2-snapshot-java11",
							}},
						},
						"components": {
							FieldID: "components",
							AllowedValues: []jira.IssueTypeMetaAllowedValue{{
								Name: "java11",
							}},
						},
					}, nil)
				m.On("CreateComponentValue", testify.Anything, 1, "java11-mvn-create").
					Return(nil).
					Maybe()

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "wrong type of payload value")
			},
		},
		{
			name: "failed to get issue type meta",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(&goJira.Issue{
						Fields: &goJira.IssueFields{
							Type: goJira.IssueType{
								ID:   "2",
								Name: "TEST-1",
							},
						},
					}, nil)
				m.On("GetIssueTypeMeta", testify.Anything, "1", "2").
					Return(nil, errors.New("failed"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get issue type meta")
			},
		},
		{
			name: "failed to get issue type",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(&goJira.Project{
						ID: "1",
					}, nil)
				m.On("GetIssue", testify.Anything, "TEST-1").
					Return(nil, errors.New("failed"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get issue")
			},
		},
		{
			name: "failed to get project info",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(nil, errors.New("failed"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get project info")
			},
		},
		{
			name: "project info not found",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				m := jiraMock.NewMockClient(t)
				m.On("GetProjectInfo", "TEST-1").
					Return(nil, fmt.Errorf("project not found %w", jira.ErrNotFound))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "jira issue not found")
			},
		},
		{
			name: "invalid payload",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{"TEST-1"},
					Payload: "{{{{{",
				},
			},
			client: func(t *testing.T) jira.Client {
				return jiraMock.NewMockClient(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get map with Jira field values")
			},
		},
		{
			name: "no tickets",
			metadata: &codebaseApi.JiraIssueMetadata{
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Tickets: []string{},
					Payload: "{\"components\":\"java11-mvn-create\",\"fixVersions\":\"0.0.1-snapshot-java11\",\"labels\":\"java11\",\"issuesLinks\":[{\"ticket\":\"EPMDEDPSUP-5510\",\"title\":\"[EPMDEDPSUP-5510]: test\",\"url\": \"https://test.com\"}]}",
				},
			},
			client: func(t *testing.T) jira.Client {
				return jiraMock.NewMockClient(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "JiraIssueMetadata is invalid. Tickets field can't be empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := PutTagValue{
				client: tt.client(t),
			}

			tt.wantErr(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.metadata))
		})
	}
}
