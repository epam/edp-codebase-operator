package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type CodebaseHandler interface {
	ServeRequest(context.Context, *codebaseApi.Codebase) error
}
