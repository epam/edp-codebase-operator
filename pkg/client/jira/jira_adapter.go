package jira

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	url "net/url"
	"strconv"
	"strings"

	"github.com/andygrunwald/go-jira"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("gojira_adapter")

var ErrNotFound = errors.New("404")

// IssueTypeMeta represents issue metadata response from Jira.
// It is not full representation of response, only fields that are used in codebase-operator.
// See https://docs.atlassian.com/software/jira/docs/api/REST/9.4.5/#api/2/issue-getCreateIssueMetaFields.
type IssueTypeMeta struct {
	FieldID       string                      `json:"fieldId,omitempty"`
	AllowedValues []IssueTypeMetaAllowedValue `json:"allowedValues,omitempty"`
}

// IssueTypeMetaAllowedValue represents allowed value for issue type metadata response from Jira.
type IssueTypeMetaAllowedValue struct {
	Name string `json:"name,omitempty"`
}

// List represents pagination response from Jira.
type List[T any] struct {
	MaxResults int  `json:"maxResults"`
	StartAt    int  `json:"startAt"`
	Total      int  `json:"total"`
	IsLat      bool `json:"isLast"`
	Values     []T  `json:"values"`
}

type GoJiraAdapter struct {
	client jira.Client
}

func (a *GoJiraAdapter) Connected() (bool, error) {
	log.V(2).Info("start Connected method")

	ctx := context.Background()

	user, _, err := a.client.User.GetSelfWithContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to fetch jira user: %w", err)
	}

	return user != nil, nil
}

func (a *GoJiraAdapter) GetIssue(ctx context.Context, issueId string) (*jira.Issue, error) {
	issue, _, err := a.client.Issue.GetWithContext(ctx, issueId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jira issue: %w", err)
	}

	return issue, nil
}

func (a *GoJiraAdapter) GetProjectInfo(issueId string) (*jira.Project, error) {
	logv := log.WithValues("issue", issueId)
	logv.V(2).Info("start GetProjectInfo method.")

	ctx := context.Background()

	issueResp, httpResp, err := a.client.Issue.GetWithContext(ctx, issueId, nil)

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to fetch jira issue: %w", err)
	}

	logv.V(2).Info("project info has been fetched.", "id", issueResp.Fields.Project.ID)

	return &issueResp.Fields.Project, nil
}

func (a *GoJiraAdapter) CreateFixVersionValue(ctx context.Context, projectId int, versionName string) error {
	logv := ctrl.LoggerFrom(ctx).WithValues("project id", projectId, "version name", versionName)
	logv.Info("Start CreateFixVersionValue method")

	_, _, err := a.client.Version.CreateWithContext(ctx, &jira.Version{
		Name:      versionName,
		ProjectID: projectId,
	})
	if err != nil {
		return fmt.Errorf("failed to create jira version component: %w", err)
	}

	logv.Info("Fix version has been created")

	return nil
}

func (a *GoJiraAdapter) CreateComponentValue(ctx context.Context, projectId int, componentName string) error {
	logv := ctrl.LoggerFrom(ctx).WithValues("project id", projectId, "component name", componentName)
	logv.Info("Start CreateComponentValue method")

	project, _, err := a.client.Project.Get(strconv.Itoa(projectId))
	if err != nil {
		return fmt.Errorf("failed to fetch jira project: %w", err)
	}

	_, _, err = a.client.Component.CreateWithContext(ctx, &jira.CreateComponentOptions{
		Name:      componentName,
		Project:   project.Key,
		ProjectID: projectId,
	})
	if err != nil {
		return fmt.Errorf("failed to create jira component: %w", err)
	}

	logv.Info("Component value has been created")

	return nil
}

func (a *GoJiraAdapter) ApplyTagsToIssue(issue string, tags map[string]interface{}) error {
	logv := log.WithValues("issue", issue, "tags", tags)
	logv.V(2).Info("start ApplyTagsToIssue method.")

	ctx := context.Background()

	if _, err := a.client.Issue.UpdateIssueWithContext(ctx, issue, tags); err != nil {
		return fmt.Errorf("failed to update jira issue: %w", err)
	}

	logv.Info("end ApplyTagsToIssue method.")

	return nil
}

func (a *GoJiraAdapter) CreateIssueLink(issueId, title, link string) error {
	logv := log.WithValues("issueId", issueId, "title", title, "url", link)
	logv.V(2).Info("start CreateIssueLink method.")

	ctx := context.Background()
	remoteLink := &jira.RemoteLink{
		Relationship: "links to",
		Object: &jira.RemoteLinkObject{
			Title: title,
			URL:   link,
			Icon: &jira.RemoteLinkIcon{
				Url16x16: "https://raw.githubusercontent.com/KubeRocketCI/docs/main/static/img/kuberocketci.png",
			},
		},
	}

	req, err := a.client.NewRequestWithContext(ctx, "POST", fmt.Sprintf("rest/api/2/issue/%v/remotelink", issueId), remoteLink)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request to jira: %w", err)
	}

	_, err = a.client.Do(req, nil)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request to jira: %w", err)
	}

	log.Info("end CreateIssueLink method.")

	return nil
}

// GetIssueTypeMeta returns issue type meta for given project and issue type.
// Map key is issue type meta fieldId, value is IssueTypeMeta.
// API doc: https://docs.atlassian.com/software/jira/docs/api/REST/9.4.5/#api/2/issue-getCreateIssueMetaFields.
func (a *GoJiraAdapter) GetIssueTypeMeta(ctx context.Context, projectID, issueTypeID string) (map[string]IssueTypeMeta, error) {
	u := url.URL{
		Path: fmt.Sprintf("rest/api/2/issue/createmeta/%s/issuetypes/%s", projectID, issueTypeID),
	}

	uv := url.Values{}
	// it is crucial to set maxResults to a high value, otherwise we will not get all the fields(default is 50)
	uv.Add("maxResults", "1000")

	req, err := a.client.NewRequestWithContext(
		ctx,
		http.MethodGet,
		u.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GetIssueTypeMeta HTTP request to jira: %w", err)
	}

	issueTypeMeta := &List[IssueTypeMeta]{}
	if _, err = a.client.Do(req, issueTypeMeta); err != nil {
		return nil, fmt.Errorf("failed to perform GetIssueTypeMeta HTTP request to jira: %w", err)
	}

	issueTypeMetaMap := make(map[string]IssueTypeMeta, len(issueTypeMeta.Values))
	for _, issueMeta := range issueTypeMeta.Values {
		issueTypeMetaMap[issueMeta.FieldID] = issueMeta
	}

	return issueTypeMetaMap, nil
}
