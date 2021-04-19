package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/imagestreamtag/chain/handler"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("image_stream_tag_handler")

func CreateDefChain(client client.Client) handler.ImageStreamTagHandler {
	return PutTagCodebaseImageStreamCr{
		client: client,
		next: DeleteTagCodebaseImageStreamCr{
			client: client,
		},
	}
}

func nextServeOrNil(next handler.ImageStreamTagHandler, ist *v1alpha1.ImageStreamTag) error {
	if next != nil {
		return next.ServeRequest(ist)
	}
	log.Info("handling of ImageStreamTag has been finished", "name", ist.Name)
	return nil
}
