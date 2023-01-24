package adapter

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/andygrunwald/go-jira"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("gojira_adapter")

var ErrNotFound = errors.New("404")

type GoJiraAdapter struct {
	client jira.Client
}

func (a GoJiraAdapter) Connected() (bool, error) {
	log.V(2).Info("start Connected method")
	user, _, err := a.client.User.GetSelf()
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

func (a GoJiraAdapter) GetIssueMetadata(projectKey string) (*jira.CreateMetaInfo, error) {
	logv := log.WithValues("projectKey", projectKey)
	logv.V(2).Info("start GetIssueMetadata method.")
	meta, _, err := a.client.Issue.GetCreateMeta(projectKey)
	if err != nil {
		return nil, err
	}
	logv.Info("end GetIssueMetadata method.")
	return meta, nil
}

func (a GoJiraAdapter) GetIssueType(issueId string) (string, error) {
	logv := log.WithValues("issueId", issueId)
	logv.V(2).Info("start GetIssueType method.")
	issue, _, err := a.client.Issue.Get(issueId, nil)
	if err != nil {
		return "", err
	}
	logv.Info("end GetIssueType method.")
	return issue.Fields.Type.Name, nil
}

func (a GoJiraAdapter) GetProjectInfo(issueId string) (*jira.Project, error) {
	logv := log.WithValues("issue", issueId)
	logv.V(2).Info("start GetProjectInfo method.")
	issueResp, httpResp, err := a.client.Issue.Get(issueId, nil)
	if httpResp != nil && httpResp.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, ErrNotFound
		}

		return nil, err
	}
	logv.V(2).Info("project info has been fetched.", "id", issueResp.Fields.Project.ID)
	return &issueResp.Fields.Project, nil
}

func (a GoJiraAdapter) CreateFixVersionValue(projectId int, versionName string) error {
	logv := log.WithValues("project id", projectId, "version name", versionName)
	logv.V(2).Info("start CreateFixVersionValue method.")
	_, _, err := a.client.Version.Create(&jira.Version{
		Name:      versionName,
		ProjectID: projectId,
	})
	if err != nil {
		return err
	}
	logv.Info("fix version has been created.")
	return nil
}

func (a GoJiraAdapter) CreateComponentValue(projectId int, componentName string) error {
	logv := log.WithValues("project id", projectId, "version name", componentName)
	logv.V(2).Info("start CreateComponentValue method.")
	project, _, err := a.client.Project.Get(strconv.Itoa(projectId))
	if err != nil {
		return err
	}

	_, _, err = a.client.Component.Create(&jira.CreateComponentOptions{
		Name:      componentName,
		Project:   project.Key,
		ProjectID: projectId,
	})
	if err != nil {
		return err
	}
	logv.Info("component value has been created.")
	return nil
}

func (a GoJiraAdapter) ApplyTagsToIssue(issue string, tags map[string]interface{}) error {
	logv := log.WithValues("issue", issue, "tags", tags)
	logv.V(2).Info("start ApplyTagsToIssue method.")
	if _, err := a.client.Issue.UpdateIssue(issue, tags); err != nil {
		return err
	}
	logv.Info("end ApplyTagsToIssue method.")
	return nil
}

func (a GoJiraAdapter) CreateIssueLink(issueId, title, url string) error {
	logv := log.WithValues("issueId", issueId, "title", title, "url", url)
	logv.V(2).Info("start CreateIssueLink method.")
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
	req, err := a.client.NewRequest("POST", fmt.Sprintf("rest/api/2/issue/%v/remotelink", issueId), remoteLink)
	if err != nil {
		return err
	}

	_, err = a.client.Do(req, nil)
	if err != nil {
		return err
	}
	log.Info("end CreateIssueLink method.")
	return nil
}

func getUrlFromUri(uri string) string {
	return regexp.MustCompile(`^(?:https?:\/\/)?(?:[^@\/\n]+@)?(?:www\.)?([^:\/\n]+)`).FindAllString(uri, -1)[0]
}
