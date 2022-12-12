package handler

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type JiraServerHandler interface {
	ServeRequest(jira *codebaseApi.JiraServer) error
}
