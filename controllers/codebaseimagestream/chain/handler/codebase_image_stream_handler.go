package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type CodebaseImageStreamHandler interface {
	ServeRequest(ctx context.Context, imageStream *codebaseApi.CodebaseImageStream) error
}
