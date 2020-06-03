package chain

import (
	"context"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jirafixversion/chain/handler"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteFixVersionCr struct {
	next handler.JiraFixVersionHandler
	jc   jira.Client
	c    client.Client
}

func (h DeleteFixVersionCr) ServeRequest(version *v1alpha1.JiraFixVersion) error {
	logv := log.WithValues("fix version", version.Name)
	logv.V(2).Info("start deleting fix version cr.")

	if err := h.c.Delete(context.TODO(), version); err != nil {
		return errors.Wrapf(err, "couldn't remove fix version cr %v.", version.Name)
	}

	logv.Info("fix version cr has been deleted.")
	return nextServeOrNil(h.next, version)
}
