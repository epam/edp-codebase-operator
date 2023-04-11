package handler

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// CDStageDeployHandler is an interface for cd stage deploy chain handlers.
//
//go:generate mockery --name CDStageDeployHandler --filename handler_mock.go
type CDStageDeployHandler interface {
	ServeRequest(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error
}
