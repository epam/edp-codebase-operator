package chain

import (
	"context"
	"fmt"
	"strings"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// PutDefaultCodeBaseBranch is a struct which implements CodebaseHandler interface.
type PutDefaultCodeBaseBranch struct {
	client client.Client
}

// NewPutDefaultCodeBaseBranch is a constructor for PutDefaultCodeBaseBranch struct.
func NewPutDefaultCodeBaseBranch(c client.Client) *PutDefaultCodeBaseBranch {
	return &PutDefaultCodeBaseBranch{client: c}
}

// ServeRequest gets the default branch from CodeBase CR and creates CodeBaseBranch CR with this branch.
func (s *PutDefaultCodeBaseBranch) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	codeBaseBranchName := fmt.Sprintf("%s-%s", codebase.Name, processNameToKubernetesConvention(codebase.Spec.DefaultBranch))

	log := ctrl.LoggerFrom(ctx).WithValues("codeBaseBranchName", codeBaseBranchName)

	branch := &codebaseApi.CodebaseBranch{}
	err := s.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: codebase.Namespace,
			Name:      codeBaseBranchName,
		},
		branch,
	)

	if err == nil {
		log.Info("Codebase branch already exists")

		return nil
	}

	if err != nil && !k8sErrors.IsNotFound(err) {
		return fmt.Errorf("failed to get codebase branch: %w", err)
	}

	branch = &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      codeBaseBranchName,
			Namespace: codebase.Namespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   codebase.Spec.DefaultBranch,
			CodebaseName: codebase.Name,
		},
	}

	if codebase.Spec.Versioning.Type != codebaseApi.VersioningTypDefault {
		branch.Spec.Version = codebase.Spec.Versioning.StartFrom
	}

	if err = s.client.Create(ctx, branch); err != nil {
		return fmt.Errorf("failed to create codebase branch: %w", err)
	}

	log.Info("Codebase branch has been created")

	return nil
}

func processNameToKubernetesConvention(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}
