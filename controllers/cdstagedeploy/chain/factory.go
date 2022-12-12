package chain

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/handler"
)

func CreateDefChain(c client.Client) handler.CDStageDeployHandler {
	return PutCDStageJenkinsDeployment{
		client: c,
		log:    ctrl.Log.WithName("put-cd-stage-jenkins-deployment-controller"),
	}
}
