package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gittag/chain/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("git_tag_handler")

func CreateDefChain(client client.Client) handler.GitTagHandler {
	return PushGitTag{
		client: client,
		next: DeleteGitTagCr{
			client: client,
		},
		git: gitserver.GitProvider{},
	}
}

func nextServeOrNil(next handler.GitTagHandler, gt *v1alpha1.GitTag) error {
	if next != nil {
		return next.ServeRequest(gt)
	}
	log.Info("handling of GitTag has been finished", "name", gt.Name)
	return nil
}
