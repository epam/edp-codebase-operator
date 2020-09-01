package chain

import (
	"context"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/imagestreamtag/chain/handler"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteTagCodebaseImageStreamCr struct {
	next   handler.ImageStreamTagHandler
	client client.Client
}

func (h DeleteTagCodebaseImageStreamCr) ServeRequest(ist *v1alpha1.ImageStreamTag) error {
	rl := log.WithValues("image stream tag name", ist.Name)
	rl.Info("start DeleteTagCodebaseImageStreamCr chain executing...")
	ist, err := h.get(ist.Name, ist.Namespace)
	if err != nil {
		return err
	}

	if err := h.delete(ist); err != nil {
		return err
	}

	rl.Info("end DeleteTagCodebaseImageStreamCr chain executing...")
	return nextServeOrNil(h.next, ist)
}

func (h DeleteTagCodebaseImageStreamCr) get(name, namespace string) (*v1alpha1.ImageStreamTag, error) {
	ist := &v1alpha1.ImageStreamTag{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, ist)
	if err != nil {
		return nil, err
	}
	return ist, nil
}

func (h DeleteTagCodebaseImageStreamCr) delete(tag *v1alpha1.ImageStreamTag) error {
	if err := h.client.Delete(context.TODO(), tag); err != nil {
		return errors.Wrapf(err, "couldn't remove image stream tag %v.", tag.Name)
	}
	log.Info("image stream tag has been removed", "name", tag.Name)
	return nil
}
