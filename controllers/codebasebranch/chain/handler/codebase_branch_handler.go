package handler

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type CodebaseBranchHandler interface {
	ServeRequest(ctx context.Context, c *codebaseApi.CodebaseBranch) error
}

var log = ctrl.Log.WithName("codebase_branch_handler")

func NextServeOrNil(ctx context.Context, next CodebaseBranchHandler, cb *codebaseApi.CodebaseBranch) error {
	if next == nil {
		log.Info("handling of codebase branch has been finished", "name", cb.Name)

		return nil
	}

	err := next.ServeRequest(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to server handler in a chain: %w", err)
	}

	return nil
}
