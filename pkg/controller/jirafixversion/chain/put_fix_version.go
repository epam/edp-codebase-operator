package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jirafixversion/chain/handler"
	"github.com/pkg/errors"
	"strconv"
)

type PutFixVersion struct {
	next   handler.JiraFixVersionHandler
	client jira.Client
}

func (h PutFixVersion) ServeRequest(version *v1alpha1.JiraFixVersion) error {
	logv := log.WithValues("fix version", version.Name)
	logv.V(2).Info("start putting fix version in project.")
	projectId, err := h.client.GetProjectId(version.Spec.Tickets[0])
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting project id by %v issue.", projectId)
	}

	exist, err := h.client.FixVersionExists(*projectId, version.Name)
	if err != nil {
		return errors.Wrapf(err, "couldn't check fix version in project with %v id.", projectId)
	}

	if exist {
		logv.Info("fix version already exists. skip creating.")
		return nextServeOrNil(h.next, version)
	}

	id, err := strconv.Atoi(*projectId)
	if err != nil {
		return err
	}

	if err := h.client.CreateFixVersion(id, version.Name); err != nil {
		return errors.Wrapf(err, "couldn't create fix version %v", version.Name)
	}
	logv.V(2).Info("end putting fix version in project.")
	return nextServeOrNil(h.next, version)
}
