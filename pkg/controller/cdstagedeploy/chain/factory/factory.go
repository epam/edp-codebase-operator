package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy/chain/putcdstagejenkinsdeployment"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDefChain(client client.Client) handler.CDStageDeployHandler {
	return putcdstagejenkinsdeployment.PutCDStageJenkinsDeployment{
		Client: client,
	}
}
