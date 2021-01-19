package jira

import (
	"github.com/andygrunwald/go-jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
)

type Client interface {
	Connected() (bool, error)

	GetProjectInfo(issueId string) (*jira.Project, error)

	CreateFixVersionValue(projectId int, versionName string) error

	CreateComponentValue(projectId int, componentName string) error

	GetIssueMetadata(projectKey string) (*jira.CreateMetaInfo, error)

	GetIssueType(issueId string) (*string, error)

	ApplyTagsToIssue(issue string, tags map[string]interface{}) error

	CreateIssueLink(issueId, title, url string) error
}

type ClientFactory interface {
	New(jira dto.JiraServer) (Client, error)
}
