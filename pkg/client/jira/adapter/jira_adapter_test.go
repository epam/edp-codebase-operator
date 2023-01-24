package adapter

import (
	"strings"
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
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
		t.Fatal("Unable to create Jira Client")
	}
	c, err := jc.Connected()
	assert.NoError(t, err)
	assert.True(t, c)
}

func TestGoJiraAdapter_Connected_False(t *testing.T) {
	httpmock.Reset()
	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
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

func TestGoJiraAdapter_GetIssueMetadata_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jm := jira.CreateMetaInfo{
		Expand: "expand",
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/issue/createmeta?expand=projects.issuetypes.fields&projectKeys=project_key",
		httpmock.NewJsonResponderOrPanic(200, &jm))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
	}
	meta, err := jc.GetIssueMetadata("project_key")
	assert.NoError(t, err)
	assert.Equal(t, meta.Expand, "expand")
}

func TestGoJiraAdapter_GetIssueMetadata_Fail(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
	}
	meta, err := jc.GetIssueMetadata("issueId")
	assert.Error(t, err)
	assert.Nil(t, meta)
}

func TestGoJiraAdapter_GetIssueType_Pass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	ji := jira.Issue{
		Expand: "expand",
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				Name: "bug",
			},
		},
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/issue/issueId",
		httpmock.NewJsonResponderOrPanic(200, &ji))

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
	}
	issue, err := jc.GetIssueType("issueId")
	assert.NoError(t, err)
	assert.Equal(t, issue, "bug")
}

func TestGoJiraAdapter_GetIssueType_Fail(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
	}
	issue, err := jc.GetIssueType("issueId")
	assert.Error(t, err)
	assert.Equal(t, "", issue)
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
		t.Fatal("Unable to create Jira Client")
	}
	jp, err := jc.GetProjectInfo("issueId")
	assert.NoError(t, err)
	assert.Equal(t, jp.Name, "test")
}

func TestGoJiraAdapter_GetProjectInfo_Fail(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
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
		t.Fatal("Unable to create Jira Client")
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/issue/issueId",
		httpmock.NewStringResponder(404, "not found"))

	_, err = jc.GetProjectInfo("issueId")
	assert.ErrorIs(t, err, ErrNotFound)

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/issue/issueId",
		httpmock.NewStringResponder(200, "not found: 404"))

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
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateFixVersionValue(1, "100")
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
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateFixVersionValue(1, "100")
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
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateComponentValue(1, "100")
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
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateComponentValue(1, "100")
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
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateComponentValue(1, "100")
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
		t.Fatal("Unable to create Jira Client")
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
		t.Fatal("Unable to create Jira Client")
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
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateIssueLink("jiraId", "title", "url")
	assert.NoError(t, err)
}

func TestGoJiraAdapter_CreateIssueLink_Fail(t *testing.T) {
	httpmock.Reset()

	jc, err := new(GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer("j-api", "user", "pwd"))
	if err != nil {
		t.Fatal("Unable to create Jira Client")
	}
	err = jc.CreateIssueLink("jiraId", "title", "url")
	assert.Error(t, err)
}
