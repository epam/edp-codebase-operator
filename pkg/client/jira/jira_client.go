package jira

import (
	"context"

	"github.com/andygrunwald/go-jira"

	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
)

// Client is an interface that describes methods for working with Jira.
type Client interface {
	Connected() (bool, error)

	GetProjectInfo(issueId string) (*jira.Project, error)

	CreateFixVersionValue(ctx context.Context, projectId int, versionName string) error

	CreateComponentValue(ctx context.Context, projectId int, componentName string) error

	GetIssue(ctx context.Context, issueId string) (*jira.Issue, error)

	ApplyTagsToIssue(issue string, tags map[string]interface{}) error

	CreateIssueLink(issueId, title, url string) error

	GetIssueTypeMeta(ctx context.Context, projectID, issueTypeID string) (map[string]IssueTypeMeta, error)
}

type ClientFactory interface {
	New(jira dto.JiraServer) (Client, error)
}
