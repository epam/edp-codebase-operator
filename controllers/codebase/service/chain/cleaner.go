package chain

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (h *Cleaner) ServeRequest(_ context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("start cleaning data...")

	if err := h.tryToClean(c); err != nil {
		setFailedFields(c, codebaseApi.CleanData, err.Error())

		return err
	}

	rLog.Info("end cleaning data...")

	return nil
}

func (h *Cleaner) tryToClean(c *codebaseApi.Codebase) error {
	s := fmt.Sprintf("repository-codebase-%v-temp", c.Name)

	if err := h.deleteSecret(s, c.Namespace); err != nil {
		return fmt.Errorf("failed to delete secret %v: %w", s, err)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)

	return deleteWorkDirectory(wd)
}

func (h *Cleaner) deleteSecret(secretName, namespace string) error {
	log.Info("start deleting secret", "name", secretName)

	ctx := context.Background()

	if err := h.client.Delete(ctx, &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("secret doesn't exist. skip deleting", "name", secretName)

			return nil
		}

		return fmt.Errorf("failed to Delete 'Secret' resource %q: %w", secretName, err)
	}

	log.Info("end deleting secret", "name", secretName)

	return nil
}

func deleteWorkDirectory(dir string) error {
	if err := util.RemoveDirectory(dir); err != nil {
		return fmt.Errorf("failed to delete directory %v: %w", dir, err)
	}

	log.Info("directory was cleaned", "path", dir)

	return nil
}
