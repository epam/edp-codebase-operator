package chain

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/handler"
)

type CDStageDeployChain func(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error)

func CreateDefChain(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error) {
	codebase := &codebaseApi.Codebase{}
	if err := cl.Get(ctx,
		types.NamespacedName{
			Name:      deploy.Spec.Tag.Codebase,
			Namespace: deploy.Namespace,
		}, codebase,
	); err != nil {
		return nil, fmt.Errorf("failed to get codebase: %w", err)
	}

	return &UpdateArgoApplicationTag{
		client: cl,
		next: &DeleteCDStageDeploy{
			client: cl,
		},
	}, nil
}

func nextServeOrNil(ctx context.Context, next handler.CDStageDeployHandler, deploy *codebaseApi.CDStageDeploy) error {
	if next == nil {
		return nil
	}

	if err := next.ServeRequest(ctx, deploy); err != nil {
		return fmt.Errorf("failed to process handler in a chain: %w", err)
	}

	return nil
}
