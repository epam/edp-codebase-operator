package chain

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/imagestreamtag/chain/handler"
)

var log = ctrl.Log.WithName("image_stream_tag_handler")

func CreateDefChain(c client.Client) handler.ImageStreamTagHandler {
	return PutTagCodebaseImageStreamCr{
		client: c,
		next: DeleteTagCodebaseImageStreamCr{
			client: c,
		},
	}
}

func nextServeOrNil(next handler.ImageStreamTagHandler, ist *codebaseApi.ImageStreamTag) error {
	if next == nil {
		log.Info("handling of ImageStreamTag has been finished", "name", ist.Name)

		return nil
	}

	err := next.ServeRequest(ist)
	if err != nil {
		return fmt.Errorf("failed to process handler in a chain: %w", err)
	}

	return nil
}
