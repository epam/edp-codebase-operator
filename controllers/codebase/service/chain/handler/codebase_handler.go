package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// CodebaseHandler is an interface for codebase chain handlers.
type CodebaseHandler interface {
	ServeRequest(context.Context, *codebaseApi.Codebase) error
}
