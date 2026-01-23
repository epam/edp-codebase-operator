package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/go-logr/logr"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
)

func TestGoJiraAdapter_Connected_True(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	ju := jira.User{
		Name: "user",
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/myself",
		httpmock.NewJsonResponderOrPanic(200, &ju))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	c, err := jc.Connected()

	assert.NoError(t, err)
	assert.True(t, c)
}

func TestGoJiraAdapter_Connected_False(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	c, err := jc.Connected()

	assert.Error(t, err)
	assert.False(t, c)
}

func TestGoJiraAdapter_UnableCreateJiraClient(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("htt\\p://", "user", "pwd"))

	assert.Error(t, err)
	assert.Nil(t, jc)

	if !strings.Contains(err.Error(), "parse \"htt\\\\p://\"") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoJiraAdapter_GetProjectInfo_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	ji := jira.Issue{
		Expand: "expand",
		Fields: &jira.IssueFields{
			Project: jira.Project{
				Name: "test",
			},
		},
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/issue/issueId",
		httpmock.NewJsonResponderOrPanic(200, &ji))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	jp, err := jc.GetProjectInfo("issueId")

	assert.NoError(t, err)
	assert.Equal(t, jp.Name, "test")
}

func TestGoJiraAdapter_GetProjectInfo_Fail(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	jp, err := jc.GetProjectInfo("issueId")

	assert.Error(t, err)
	assert.Nil(t, jp)
}

func TestGoJiraAdapter_GetProjectInfo_Fail_IssueNotFound(t *testing.T) {
	httpmock.Reset()

	httpmock.Activate()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	httpmock.RegisterResponder(
		"GET",
		"/j-api/rest/api/2/issue/issueId",
		func(*http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(404, "not found"), nil
		},
	)

	_, err = jc.GetProjectInfo("issueId")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGoJiraAdapter_CreateFixVersionValue_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jv := jira.Version{}

	httpmock.RegisterResponder("POST", "/j-api/rest/api/2/version",
		httpmock.NewJsonResponderOrPanic(200, &jv))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateFixVersionValue(ctrl.LoggerInto(context.Background(), logr.Discard()), 1, "100")

	assert.NoError(t, err)
}

func TestGoJiraAdapter_CreateFixVersionValue_Fail(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jv := jira.Error{}

	httpmock.RegisterResponder("POST", "/j-api/rest/api/2/version",
		httpmock.NewJsonResponderOrPanic(404, &jv))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateFixVersionValue(ctrl.LoggerInto(context.Background(), logr.Discard()), 1, "100")

	assert.Error(t, err)
}

func TestGoJiraAdapter_CreateComponentValue_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jv := jira.Component{}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/project/1",
		httpmock.NewJsonResponderOrPanic(200, &jv))

	httpmock.RegisterResponder("POST", "/j-api/rest/api/2/component",
		httpmock.NewJsonResponderOrPanic(200, &jv))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateComponentValue(ctrl.LoggerInto(context.Background(), logr.Discard()), 1, "100")

	assert.NoError(t, err)
}

func TestGoJiraAdapter_CreateComponentValue_FailToGetProject(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jv := jira.Error{}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/project/1",
		httpmock.NewJsonResponderOrPanic(404, &jv))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateComponentValue(ctrl.LoggerInto(context.Background(), logr.Discard()), 1, "100")

	assert.Error(t, err)
}

func TestGoJiraAdapter_CreateComponentValue_FailToCreateComponent(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jv := jira.Project{}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/project/1",
		httpmock.NewJsonResponderOrPanic(200, &jv))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateComponentValue(ctrl.LoggerInto(context.Background(), logr.Discard()), 1, "100")

	assert.Error(t, err)
}

func TestGoJiraAdapter_ApplyTagsToIssue_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jv := jira.Issue{}
	params := map[string]interface{}{
		"update": "test",
	}

	httpmock.RegisterResponder("PUT", "/j-api/rest/api/2/issue/jiraId",
		httpmock.NewJsonResponderOrPanic(200, &jv))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.ApplyTagsToIssue("jiraId", params)

	assert.NoError(t, err)
}

func TestGoJiraAdapter_ApplyTagsToIssue_Fail(t *testing.T) {
	httpmock.Reset()

	params := map[string]interface{}{
		"update": "test",
	}

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.ApplyTagsToIssue("jiraId", params)

	assert.Error(t, err)
}

func TestGoJiraAdapter_CreateIssueLink_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jrl := jira.RemoteLink{}

	httpmock.RegisterResponder("POST", "/j-api/rest/api/2/issue/jiraId/remotelink",
		httpmock.NewJsonResponderOrPanic(200, &jrl))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateIssueLink("jiraId", "title", "url")

	assert.NoError(t, err)
}

func TestGoJiraAdapter_CreateIssueLink_Fail(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("failed to create Jira Client")
	}

	err = jc.CreateIssueLink("jiraId", "title", "url")

	assert.Error(t, err)
}

func TestGoJiraAdapter_GetIssue(t *testing.T) {
	httpmock.DeactivateAndReset()

	tests := []struct {
		name     string
		httpResp http.HandlerFunc
		wantErr  require.ErrorAssertionFunc
		want     *jira.Issue
	}{
		{
			name: "successfully get issue",
			httpResp: func(w http.ResponseWriter, r *http.Request) {
				jsonResp, _ := json.Marshal(&jira.Issue{
					ID: "123",
				})

				_, _ = w.Write(jsonResp)
			},
			wantErr: require.NoError,
			want: &jira.Issue{
				ID: "123",
			},
		},
		{
			name: "failed to get issue",
			httpResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to fetch jira issue")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.httpResp)
			defer ts.Close()

			client, err := GoJiraAdapterFactory{}.New(
				dto.ConvertSpecToJiraServer(ts.URL, "user", "pwd"),
			)
			require.NoError(t, err)

			got, err := client.GetIssue(ctrl.LoggerInto(context.Background(), logr.Discard()), "1")
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoJiraAdapter_GetIssueTypeMeta(t *testing.T) {
	httpmock.DeactivateAndReset()

	tests := []struct {
		name     string
		httpResp http.HandlerFunc
		wantErr  require.ErrorAssertionFunc
		want     map[string]IssueTypeMeta
	}{
		{
			name: "successfully get issue",
			httpResp: func(w http.ResponseWriter, r *http.Request) {
				jsonResp, _ := json.Marshal(&List[IssueTypeMeta]{
					MaxResults: 1,
					StartAt:    0,
					Total:      1,
					IsLat:      true,
					Values: []IssueTypeMeta{{
						FieldID: "fixVersions",
						AllowedValues: []IssueTypeMetaAllowedValue{{
							Name: "1.0.0",
						}},
					}},
				})

				_, _ = w.Write(jsonResp)
			},
			wantErr: require.NoError,
			want: map[string]IssueTypeMeta{
				"fixVersions": {
					FieldID: "fixVersions",
					AllowedValues: []IssueTypeMetaAllowedValue{{
						Name: "1.0.0",
					}},
				},
			},
		},
		{
			name: "failed to get issue",
			httpResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to perform GetIssueTypeMeta HTTP request to jira")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.httpResp)
			defer ts.Close()

			client, err := GoJiraAdapterFactory{}.New(
				dto.ConvertSpecToJiraServer(ts.URL, "user", "pwd"),
			)
			require.NoError(t, err)

			got, err := client.GetIssueTypeMeta(ctrl.LoggerInto(context.Background(), logr.Discard()), "1", "2")
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
