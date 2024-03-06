package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// DeleteCDStageDeploy is a handler that deletes CDStageDeploy.
type DeleteCDStageDeploy struct {
	client client.Client
}

func NewDeleteCDStageDeploy(k8sClient client.Client) *DeleteCDStageDeploy {
	return &DeleteCDStageDeploy{client: k8sClient}
}

// ServeRequest handles the request to delete CDStageDeploy.
func (h *DeleteCDStageDeploy) ServeRequest(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Deleting CDStageDeploy")

	if err := client.IgnoreNotFound(h.client.Delete(ctx, stageDeploy)); err != nil {
		return fmt.Errorf("failed to delete CDStageDeploy: %w", err)
	}

	log.Info("CDStageDeploy has been deleted")

	return nil
}
