package handler

import codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"

type GitTagHandler interface {
	ServeRequest(jira *codebaseApi.GitTag) error
}
