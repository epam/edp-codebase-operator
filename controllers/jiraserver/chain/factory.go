package chain

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraserver/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
)

var log = ctrl.Log.WithName("jira_server_handler")

func CreateDefChain(jc jira.Client, c client.Client) handler.JiraServerHandler {
	return CheckConnection{
		next: PutJiraEDPComponent{
			next:   nil,
			client: c,
		},
		client: jc,
	}
}

func nextServeOrNil(next handler.JiraServerHandler, js *codebaseApi.JiraServer) error {
	if next == nil {
		log.Info("handling of JiraServer has been finished", "jira server name", js.Name)
		return nil
	}

	err := next.ServeRequest(js)
	if err != nil {
		return fmt.Errorf("failed to process next handler in a chain: %w", err)
	}

	return nil
}
