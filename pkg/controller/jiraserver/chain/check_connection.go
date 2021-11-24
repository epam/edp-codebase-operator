package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraserver/chain/handler"
	"github.com/pkg/errors"
)

type CheckConnection struct {
	next   handler.JiraServerHandler
	client jira.Client
}

func (h CheckConnection) ServeRequest(jira *v1alpha1.JiraServer) error {
	rl := log.WithValues("jira server name", jira.Name)
	rl.V(2).Info("start checking connection...")
	connected, err := h.checkConnection(*jira)
	jira.Status.Available = err == nil && connected
	if err != nil {
		return err
	}
	rl.Info("end checking connection...")
	return nextServeOrNil(h.next, jira)
}

func (h CheckConnection) checkConnection(jira v1alpha1.JiraServer) (bool, error) {
	connected, err := h.client.Connected()
	if err != nil {
		return false, errors.Wrap(err, "couldn't connect to Jira server")
	}
	log.Info("connection to Jira server", "established", connected)
	return connected, nil
}
