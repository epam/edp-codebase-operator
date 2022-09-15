package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type Cleaner struct {
	client client.Client
}

func NewCleaner(client client.Client) *Cleaner {
	return &Cleaner{client: client}
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
		return errors.Wrapf(err, "unable to delete secret %v", s)
	}
	wd := util.GetWorkDir(c.Name, c.Namespace)
	if err := deleteWorkDirectory(wd); err != nil {
		return err
	}
	return nil
}

func (h *Cleaner) deleteSecret(secretName, namespace string) error {
	log.Info("start deleting secret", "name", secretName)
	if err := h.client.Delete(context.TODO(), &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("secret doesn't exist. skip deleting", "name", secretName)
			return nil
		}
		return err
	}
	log.Info("end deleting secret", "name", secretName)
	return nil
}

func deleteWorkDirectory(dir string) error {
	if err := util.RemoveDirectory(dir); err != nil {
		return errors.Wrapf(err, "couldn't delete directory %v", dir)
	}
	log.Info("directory was cleaned", "path", dir)
	return nil
}
