package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/imagestreamtag/chain/handler"
)

type DeleteTagCodebaseImageStreamCr struct {
	next   handler.ImageStreamTagHandler
	client client.Client
}

func (h DeleteTagCodebaseImageStreamCr) ServeRequest(ist *codebaseApi.ImageStreamTag) error {
	rl := log.WithValues("image stream tag name", ist.Name)
	rl.Info("start DeleteTagCodebaseImageStreamCr chain executing...")

	if err := h.delete(ist); err != nil {
		return err
	}

	rl.Info("end DeleteTagCodebaseImageStreamCr chain executing...")

	return nextServeOrNil(h.next, ist)
}

func (h DeleteTagCodebaseImageStreamCr) delete(tag *codebaseApi.ImageStreamTag) error {
	if err := h.client.Delete(context.TODO(), tag); err != nil {
		return fmt.Errorf("failed to remove image stream tag %v: %w", tag.Name, err)
	}

	log.Info("image stream tag has been removed", "name", tag.Name)

	return nil
}
