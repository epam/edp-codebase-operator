package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var chainLog = ctrl.Log.WithName("codebase_chain")

type chain struct {
	handlers []handler.CodebaseHandler
}

func (ch *chain) Use(handlers ...handler.CodebaseHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *chain) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	chainLog.Info("starting codebase chain", "codebase_name", c.Name)

	defer util.Timer("codebase_chain", log)()

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.ServeRequest(ctx, c)
		if err != nil {
			chainLog.Info("codebase chain finished with error", "codebase_name", c.Name)

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	chainLog.Info("handling of codebase has been finished", "codebase_name", c.Name)

	return nil
}
