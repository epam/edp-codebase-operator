package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebaseimagestream/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebaseimagestream/chain/putcdstagedeploy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDefChain(client client.Client) handler.CodebaseImageStreamHandler {
	return putcdstagedeploy.PutCDStageDeploy{
		Client: client,
	}
}
