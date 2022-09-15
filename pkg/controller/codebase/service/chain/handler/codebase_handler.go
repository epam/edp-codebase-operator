package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

type CodebaseHandler interface {
	ServeRequest(context.Context, *codebaseApi.Codebase) error
}
