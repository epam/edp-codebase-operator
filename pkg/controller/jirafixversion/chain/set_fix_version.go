package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jirafixversion/chain/handler"
	"github.com/pkg/errors"
)

type SetFixVersion struct {
	next   handler.JiraFixVersionHandler
	client jira.Client
}

const successStatus = "success"

func (h SetFixVersion) ServeRequest(version *v1alpha1.JiraFixVersion) error {
	logv := log.WithValues("fix version", version.Name, "issues", version.Spec.Tickets)
	logv.V(2).Info("start setting fix version to issues.")
	if err := h.setFixVersionToTickets(version.Name, version.Spec.Tickets); err != nil {
		return errors.Wrapf(err, "couldn't set %v fix version to %v tickets",
			version.Name, version.Spec.Tickets)
	}
	version.Status.Status = successStatus
	logv.V(2).Info("end setting fix version to issues.")
	return nextServeOrNil(h.next, version)
}

func (h SetFixVersion) setFixVersionToTickets(versionName string, issues []string) error {
	for _, issue := range issues {
		if err := h.client.SetFixVersionToIssue(issue, versionName); err != nil {
			return err
		}
	}
	return nil
}
