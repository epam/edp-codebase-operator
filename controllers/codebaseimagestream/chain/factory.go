package chain

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebaseimagestream/chain/handler"
)

var log = ctrl.Log.WithName("codebase-image-stream")

func CreateDefChain(c client.Client) handler.CodebaseImageStreamHandler {
	return PutCDStageDeploy{
		client: c,
		log:    log.WithName("create-chain").WithName("put-cd-stage-deploy"),
	}
}
