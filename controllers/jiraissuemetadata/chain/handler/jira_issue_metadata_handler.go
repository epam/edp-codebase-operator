package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type JiraIssueMetadataHandler interface {
	ServeRequest(ctx context.Context, version *codebaseApi.JiraIssueMetadata) error
}
