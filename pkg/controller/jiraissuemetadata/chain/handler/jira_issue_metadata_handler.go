package handler

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

type JiraIssueMetadataHandler interface {
	ServeRequest(version *codebaseApi.JiraIssueMetadata) error
}
