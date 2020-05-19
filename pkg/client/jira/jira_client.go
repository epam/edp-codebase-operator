package jira

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/dto"
)

type Client interface {
	Connected() (bool, error)

	FixVersionExists(projectId, version string) (bool, error)

	GetProjectId(issue string) (*string, error)

	CreateFixVersion(projectId int, versionName string) error

	SetFixVersionToIssue(issue, fixVersion string) error
}

type ClientFactory interface {
	New(jira dto.JiraServer) (Client, error)
}
