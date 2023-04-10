package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// CodebaseHandler is an interface for codebase chain handlers.
//
//go:generate mockery --name CodebaseHandler --filename handler_mock.go
type CodebaseHandler interface {
	ServeRequest(context.Context, *codebaseApi.Codebase) error
}
