package k8s

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CodebaseRepository struct {
	client client.Client
	cr     *edpv1alpha1.Codebase
}

// Simple constructor for CodebaseRepository. Codebase CR is used to avoid additional calls to K8S and error
// with concurrent resource update
func NewK8SCodebaseRepository(client client.Client, cr *edpv1alpha1.Codebase) repository.CodebaseRepository {
	return CodebaseRepository{
		client: client,
		cr:     cr,
	}
}

// Retrieves status of git provisioning from codebase cr. To avoid additional call to Kubernetes, values from
// inner field codebase are used. Input parameters are codebase and edp are ignored
func (repo CodebaseRepository) SelectProjectStatusValue(codebase, edp string) (*string, error) {
	gs := repo.cr.Status.Git
	return &gs, nil
}

// Sets the input value gitStatus to the corresponding field in Codebase CR. To avoid additional call to Kubernetes,
// values from inner field codebase are used. Input parameters are codebase and edp are ignored.
func (repo CodebaseRepository) UpdateProjectStatusValue(gitStatus, codebase, edp string) error {
	repo.cr.Status.Git = gitStatus
	if err := repo.client.Status().Update(context.TODO(), repo.cr); err != nil {
		// Used for backward compatibility
		if err := repo.client.Update(context.TODO(), repo.cr); err != nil {
			return err
		}
	}
	return nil
}
