package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain/handler"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDefChain(client client.Client) handler.CDStageDeployHandler {
	return PutCDStageJenkinsDeployment{
		client: client,
		log:    ctrl.Log.WithName("put-cd-stage-jenkins-deployment-controller"),
	}
}
