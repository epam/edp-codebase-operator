package chain

import (
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jiraserver/chain/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jira_server_handler")

func CreateDefChain(jc jira.Client, client client.Client) handler.JiraServerHandler {
	return CheckConnection{
		next: PutJiraEDPComponent{
			next:   nil,
			client: client,
		},
		client: jc,
	}
}

func nextServeOrNil(next handler.JiraServerHandler, jira *edpv1alpha1.JiraServer) error {
	if next != nil {
		return next.ServeRequest(jira)
	}
	log.Info("handling of JiraServer has been finished", "jira server name", jira.Name)
	return nil
}
