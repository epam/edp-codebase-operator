package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/gittag/chain/handler"
)

type DeleteGitTagCr struct {
	next   handler.GitTagHandler
	client client.Client
}

func (h DeleteGitTagCr) ServeRequest(gt *codebaseApi.GitTag) error {
	rl := log.WithValues("gi tag name", gt.Name)
	rl.Info("start DeleteGitTagCr chain executing...")

	if err := h.delete(gt); err != nil {
		return err
	}

	rl.Info("end DeleteGitTagCr chain executing...")

	return nextServeOrNil(h.next, gt)
}

func (h DeleteGitTagCr) delete(tag *codebaseApi.GitTag) error {
	if err := h.client.Delete(context.TODO(), tag); err != nil {
		return fmt.Errorf("failed to remove git tag %v: %w", tag.Name, err)
	}

	log.Info("git tag has been removed", "name", tag.Name)

	return nil
}
