package chain

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/imagestreamtag/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutTagCodebaseImageStreamCr struct {
	next   handler.ImageStreamTagHandler
	client client.Client
}

const timePattern = "2006-01-02T15:04:05"

func (h PutTagCodebaseImageStreamCr) ServeRequest(ist *codebaseApi.ImageStreamTag) error {
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

	for _, t := range cis.Spec.Tags {
		if tag == t.Name {
			log.Info("tag already exists in CodebaseImageStream CR", "name", tag)
			return nil
		}
	}

	t, err := getCurrentTimeInUTC()
	if err != nil {
		return errors.Wrap(err, "couldn't get current time")
	}

	cis.Spec.Tags = append(cis.Spec.Tags, codebaseApi.Tag{
		Name:    tag,
		Created: *t,
	})

	return h.update(cis)
}

func getCurrentTimeInUTC() (*string, error) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, fmt.Errorf("failed to load location data: %w", err)
	}

	return util.GetStringP(time.Now().In(loc).Format(timePattern)), nil
}

func (h PutTagCodebaseImageStreamCr) getCodebaseImageStream(name, namespace string) (*codebaseApi.CodebaseImageStream, error) {
	ctx := context.Background()
	cis := &codebaseApi.CodebaseImageStream{}

	err := h.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cis)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 'CodebaseImageStream' resource %q: %w", name, err)
	}

	return cis, nil
}

func (h PutTagCodebaseImageStreamCr) update(cis *codebaseApi.CodebaseImageStream) error {
	ctx := context.Background()

	if err := h.client.Update(ctx, cis); err != nil {
		return errors.Wrapf(err, "couldn't add new tag to codebase image stream %v", cis.Name)
	}

	log.Info("cis has been updated with tag", "cis", cis.Name, "tags", cis.Spec.Tags)

	return nil
}
