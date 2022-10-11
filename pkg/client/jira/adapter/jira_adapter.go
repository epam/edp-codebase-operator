package adapter

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/andygrunwald/go-jira"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("gojira_adapter")

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

func (a *GoJiraAdapter) GetIssueMetadata(projectKey string) (*jira.CreateMetaInfo, error) {
	logv := log.WithValues("projectKey", projectKey)
	logv.V(2).Info("start GetIssueMetadata method.")

	ctx := context.Background()

	meta, _, err := a.client.Issue.GetCreateMetaWithContext(ctx, projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch/create jira metadata: %w", err)
	}

	logv.Info("end GetIssueMetadata method.")

	return meta, nil
}

func (a *GoJiraAdapter) GetIssueType(issueId string) (*string, error) {
	logv := log.WithValues("issueId", issueId)
	logv.V(2).Info("start GetIssueType method.")

	ctx := context.Background()

	issue, _, err := a.client.Issue.GetWithContext(ctx, issueId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jira issue: %w", err)
	}

	logv.Info("end GetIssueType method.")

	return util.GetStringP(issue.Fields.Type.Name), nil
}

func (a *GoJiraAdapter) GetProjectInfo(issueId string) (*jira.Project, error) {
	logv := log.WithValues("issue", issueId)
	logv.V(2).Info("start GetProjectInfo method.")

	ctx := context.Background()

	issueResp, _, err := a.client.Issue.GetWithContext(ctx, issueId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jira issue: %w", err)
	}

	logv.V(2).Info("project info has been fetched.", "id", issueResp.Fields.Project.ID)

	return &issueResp.Fields.Project, nil
}

func (a *GoJiraAdapter) CreateFixVersionValue(projectId int, versionName string) error {
	logv := log.WithValues("project id", projectId, "version name", versionName)
	logv.V(2).Info("start CreateFixVersionValue method.")

	ctx := context.Background()

	_, _, err := a.client.Version.CreateWithContext(ctx, &jira.Version{
		Name:      versionName,
		ProjectID: projectId,
	})
	if err != nil {
		return fmt.Errorf("failed to create jira version component: %w", err)
	}

	logv.Info("fix version has been created.")

	return nil
}

func (a *GoJiraAdapter) CreateComponentValue(projectId int, componentName string) error {
	logv := log.WithValues("project id", projectId, "version name", componentName)
	logv.V(2).Info("start CreateComponentValue method.")

	ctx := context.Background()

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

	logv.Info("component value has been created.")

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

func (a *GoJiraAdapter) CreateIssueLink(issueId, title, url string) error {
	logv := log.WithValues("issueId", issueId, "title", title, "url", url)
	logv.V(2).Info("start CreateIssueLink method.")

	ctx := context.Background()
	remoteLink := &jira.RemoteLink{
		Relationship: "links to",
		Object: &jira.RemoteLinkObject{
			Title: title,
			URL:   url,
			Icon: &jira.RemoteLinkIcon{
				Url16x16: fmt.Sprintf("%v/favicon.ico", getUrlFromUri(url)),
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

func getUrlFromUri(uri string) string {
	return regexp.MustCompile(`^(?:https?://)?(?:[^@/\n]+@)?(?:www\.)?([^:/\n]+)`).FindAllString(uri, -1)[0]
}
