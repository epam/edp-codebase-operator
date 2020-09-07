package chain

import (
	"context"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/imagestreamtag/chain/handler"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteTagCodebaseImageStreamCr struct {
	next   handler.ImageStreamTagHandler
	client client.Client
}

func (h DeleteTagCodebaseImageStreamCr) ServeRequest(ist *v1alpha1.ImageStreamTag) error {
	rl := log.WithValues("image stream tag name", ist.Name)
	rl.Info("start DeleteTagCodebaseImageStreamCr chain executing...")

	if err := h.delete(ist); err != nil {
		return err
	}

	rl.Info("end DeleteTagCodebaseImageStreamCr chain executing...")
	return nextServeOrNil(h.next, ist)
}

func (h DeleteTagCodebaseImageStreamCr) delete(tag *v1alpha1.ImageStreamTag) error {
	if err := h.client.Delete(context.TODO(), tag); err != nil {
		return errors.Wrapf(err, "couldn't remove image stream tag %v.", tag.Name)
	}
	log.Info("image stream tag has been removed", "name", tag.Name)
	return nil
}
