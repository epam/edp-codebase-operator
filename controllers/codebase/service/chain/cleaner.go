package chain

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type Cleaner struct {
	client client.Client
}

func NewCleaner(c client.Client) *Cleaner {
	return &Cleaner{client: c}
}

func (h *Cleaner) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start cleaning data")

	if err := h.clean(ctx, codebase); err != nil {
		setFailedFields(codebase, codebaseApi.CleanData, err.Error())

		return err
	}

	log.Info("End cleaning data")

	return nil
}

func (h *Cleaner) clean(ctx context.Context, codebase *codebaseApi.Codebase) error {
	var errs []error

	if err := h.deleteSecret(ctx, codebase); err != nil {
		errs = append(errs, err)
	}

	wd := util.GetWorkDir(codebase.Name, codebase.Namespace)

	ctrl.LoggerFrom(ctx).Info("Deleting work directory", "directory", wd)

	if err := deleteWorkDirectory(wd); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (h *Cleaner) deleteSecret(ctx context.Context, codebase *codebaseApi.Codebase) error {
	if codebase.Spec.CloneRepositoryCredentials != nil && !codebase.Spec.CloneRepositoryCredentials.ClearSecretAfterUse {
		return nil
	}

	secretName := codebase.GetCloneRepositoryCredentialSecret()
	log := ctrl.LoggerFrom(ctx).WithValues("secret", secretName)

	log.Info("Deleting secret with repository credentials for clone")

	if err := h.client.Delete(ctx, &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      secretName,
			Namespace: codebase.Namespace,
		},
	}); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Secret not found. Skipping deletion")

			return nil
		}

		return fmt.Errorf("failed to delete secret with repository credentials for clone: %w", err)
	}

	log.Info("Secret was deleted")

	return nil
}

func deleteWorkDirectory(dir string) error {
	if err := util.RemoveDirectory(dir); err != nil {
		return fmt.Errorf("failed to delete directory %v: %w", dir, err)
	}

	return nil
}
