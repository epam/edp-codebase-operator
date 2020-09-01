package chain

import (
	"context"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/imagestreamtag/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type PutTagCodebaseImageStreamCr struct {
	next   handler.ImageStreamTagHandler
	client client.Client
}

const timePattern = "2006-01-02T15:04:05"

func (h PutTagCodebaseImageStreamCr) ServeRequest(ist *v1alpha1.ImageStreamTag) error {
	rl := log.WithValues("image stream tag name", ist.Name)
	rl.Info("start PutTagCodebaseImageStreamCr chain executing...")
	if err := h.addTagToCodebaseImageStream(ist.Spec.CodebaseImageStreamName, ist.Spec.Tag, ist.Namespace); err != nil {
		return errors.Wrapf(err, "couldn't add tag to codebase image stream %v", ist.Spec.CodebaseImageStreamName)
	}
	rl.Info("end PutTagCodebaseImageStreamCr chain executing...")
	return nextServeOrNil(h.next, ist)
}

func (h PutTagCodebaseImageStreamCr) addTagToCodebaseImageStream(cisName, tag, namespace string) error {
	cis, err := h.getCodebaseImageStream(cisName, namespace)
	if err != nil {
		return err
	}

	t, err := getCurrentTimeInUTC()
	if err != nil {
		return errors.Wrap(err, "couldn't get current time")
	}

	cis.Spec.Tags = append(cis.Spec.Tags, v1alpha1.Tag{
		Name:    tag,
		Created: *t,
	})
	return h.update(cis)
}

func getCurrentTimeInUTC() (*string, error) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, err
	}
	return util.GetStringP(time.Now().In(loc).Format(timePattern)), nil
}

func (h PutTagCodebaseImageStreamCr) getCodebaseImageStream(name, namespace string) (*v1alpha1.CodebaseImageStream, error) {
	cis := &v1alpha1.CodebaseImageStream{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cis)
	if err != nil {
		return nil, err
	}
	return cis, nil
}

func (h PutTagCodebaseImageStreamCr) update(cis *v1alpha1.CodebaseImageStream) error {
	if err := h.client.Update(context.TODO(), cis); err != nil {
		return errors.Wrapf(err, "couldn't add new tag to codebase image stream %v", cis.Name)
	}
	log.Info("cis has been updated with tag", "cis", cis.Name, "tags", cis.Spec.Tags)
	return nil
}
