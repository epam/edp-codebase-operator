package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebaseimagestream/chain/handler"
)

func CreateDefChain(c client.Client) handler.CodebaseImageStreamHandler {
	return PutCDStageDeploy{
		client: c,
	}
}
