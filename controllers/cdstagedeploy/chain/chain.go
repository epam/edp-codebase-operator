package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type CDStageDeployHandler interface {
	ServeRequest(context.Context, *codebaseApi.CDStageDeploy) error
}

type chain struct {
	handlers []CDStageDeployHandler
}

func (ch *chain) Use(handlers ...CDStageDeployHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *chain) ServeRequest(ctx context.Context, c *codebaseApi.CDStageDeploy) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting CDStageDeploy chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.ServeRequest(ctx, c)
		if err != nil {
			log.Info("CDStageDeploy chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of CDStageDeploy has been finished")

	return nil
}
