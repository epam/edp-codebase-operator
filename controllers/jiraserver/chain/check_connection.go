package chain

import (
	"fmt"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraserver/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
)

type CheckConnection struct {
	next   handler.JiraServerHandler
	client jira.Client
}

func (h CheckConnection) ServeRequest(js *codebaseApi.JiraServer) error {
	rl := log.WithValues("jira server name", js.Name)
	rl.V(2).Info("start checking connection...")

	connected, err := h.checkConnection()
	if err != nil {
		return err
	}

	js.Status.Available = err == nil && connected

	rl.Info("end checking connection...")

	return nextServeOrNil(h.next, js)
}

func (h CheckConnection) checkConnection() (bool, error) {
	connected, err := h.client.Connected()
	if err != nil {
		return false, fmt.Errorf("failed to connect to Jira server: %w", err)
	}

	log.Info("connection to Jira server", "established", connected)

	return connected, nil
}
