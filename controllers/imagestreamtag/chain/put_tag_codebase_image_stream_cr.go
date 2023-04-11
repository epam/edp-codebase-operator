package chain

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/imagestreamtag/chain/handler"
)

type PutTagCodebaseImageStreamCr struct {
	next   handler.ImageStreamTagHandler
	client client.Client
}

func (h PutTagCodebaseImageStreamCr) ServeRequest(ist *codebaseApi.ImageStreamTag) error {
	rl := log.WithValues("image stream tag name", ist.Name)
	rl.Info("start PutTagCodebaseImageStreamCr chain executing...")

	if err := h.addTagToCodebaseImageStream(ist.Spec.CodebaseImageStreamName, ist.Spec.Tag, ist.Namespace); err != nil {
		return fmt.Errorf("failed to add tag to codebase image stream %v: %w", ist.Spec.CodebaseImageStreamName, err)
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

	if err != nil {
		return fmt.Errorf("failed to get current time: %w", err)
	}

	cis.Spec.Tags = append(cis.Spec.Tags, codebaseApi.Tag{
		Name:    tag,
		Created: time.Now().String(),
	})

	return h.update(cis)
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
		return fmt.Errorf("failed to add new tag to codebase image stream %v: %w", cis.Name, err)
	}

	log.Info("cis has been updated with tag", "cis", cis.Name, "tags", cis.Spec.Tags)

	return nil
}
