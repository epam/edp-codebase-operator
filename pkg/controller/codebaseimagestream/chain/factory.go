package chain

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebaseimagestream/chain/handler"
)

var log = ctrl.Log.WithName("codebase-image-stream")

func CreateDefChain(client client.Client) handler.CodebaseImageStreamHandler {
	return PutCDStageDeploy{
		client: client,
		log:    log.WithName("create-chain").WithName("put-cd-stage-deploy"),
	}
}
