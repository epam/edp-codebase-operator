package handler

import codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

type GitTagHandler interface {
	ServeRequest(jira *codebaseApi.GitTag) error
}
