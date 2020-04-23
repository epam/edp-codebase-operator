package jira

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/dto"
)

type Client interface {
	Connected() (bool, error)
}

type ClientFactory interface {
	New(jira dto.JiraServer) (Client, error)
}
