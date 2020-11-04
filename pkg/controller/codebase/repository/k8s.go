package repository

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8SCodebaseRepository struct {
	client client.Client
	cr     *edpv1alpha1.Codebase
}

// Simple constructor for K8SCodebaseRepository. Codebase CR is used to avoid additional calls to K8S and error
// with concurrent resource update
func NewK8SCodebaseRepository(client client.Client, cr *edpv1alpha1.Codebase) CodebaseRepository {
	return K8SCodebaseRepository{
		client: client,
		cr:     cr,
	}
}

// Retrieves status of git provisioning from codebase cr. To avoid additional call to Kubernetes, values from
// inner field codebase are used. Input parameters are codebase and edp are ignored
func (repo K8SCodebaseRepository) SelectProjectStatusValue(codebase, edp string) (*string, error) {
	gs := repo.cr.Status.Git
	return &gs, nil
}

// Sets the input value gitStatus to the corresponding field in Codebase CR. To avoid additional call to Kubernetes,
// values from inner field codebase are used. Input parameters are codebase and edp are ignored.
func (repo K8SCodebaseRepository) UpdateProjectStatusValue(gitStatus, codebase, edp string) error {
	repo.cr.Status.Git = gitStatus
	if err := repo.client.Status().Update(context.TODO(), repo.cr); err != nil {
		// Used for backward compatibility
		if err := repo.client.Update(context.TODO(), repo.cr); err != nil {
			return err
		}
	}
	return nil
}
