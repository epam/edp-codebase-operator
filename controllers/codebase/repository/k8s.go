package repository

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type K8SCodebaseRepository struct {
	client client.Client
	cr     *codebaseApi.Codebase
}

// Simple constructor for K8SCodebaseRepository. Codebase CR is used to avoid additional calls to K8S and error
// with concurrent resource update.
func NewK8SCodebaseRepository(c client.Client, cr *codebaseApi.Codebase) *K8SCodebaseRepository {
	return &K8SCodebaseRepository{
		client: c,
		cr:     cr,
	}
}

// Retrieves status of git provisioning from codebase cr. To avoid additional call to Kubernetes, values from
// inner field codebase are used. Input parameters are codebase and edp are ignored.
func (r *K8SCodebaseRepository) SelectProjectStatusValue(_ context.Context, _, _ string) (string, error) {
	return r.cr.Status.Git, nil
}

// Sets the input value gitStatus to the corresponding field in Codebase CR. To avoid additional call to Kubernetes,
// values from inner field codebase are used. Input parameters are codebase and edp are ignored.
func (r *K8SCodebaseRepository) UpdateProjectStatusValue(ctx context.Context, gitStatus, _, _ string) error {
	r.cr.Status.Git = gitStatus

	err := r.client.Status().Update(ctx, r.cr)
	if err != nil {
		return fmt.Errorf("failed to update status field of k8s resource: %w", err)
	}

	// Used for backward compatibility
	err = r.client.Update(ctx, r.cr)
	if err != nil {
		return fmt.Errorf("failed to update k8s resource: %w", err)
	}

	return nil
}
