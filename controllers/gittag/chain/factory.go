package chain

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/controllers/gittag/chain/handler"
)

var log = ctrl.Log.WithName("git_tag_handler")

func CreateDefChain(c client.Client) handler.GitTagHandler {
	return PushGitTag{
		client: c,
		next: DeleteGitTagCr{
			client: c,
		},
		git: &gitserver.GitProvider{},
	}
}

func nextServeOrNil(next handler.GitTagHandler, gt *codebaseApi.GitTag) error {
	if next == nil {
		log.Info("handling of GitTag has been finished", "name", gt.Name)

		return nil
	}

	err := next.ServeRequest(gt)
	if err != nil {
		return fmt.Errorf("failed to process handler in a chain: %w", err)
	}

	return nil
}
