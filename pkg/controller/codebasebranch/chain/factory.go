package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("codebase_branch_handler")

func createJenkinsDefChain(client client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "jenkins chain")
	return TriggerReleaseJob{
		client: client,
		service: service.CodebaseBranchService{
			Client: client,
		},
	}
}

func createGitlabCiDefChain(client client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "gitlab ci chain")
	return PutBranchInGit{
		client: client,
		git:    gitserver.GitProvider{},
		next: CleanTempDirectory{
			client: client,
		},
		service: service.CodebaseBranchService{
			Client: client,
		},
	}
}

func GetChain(ciType string, client client.Client) handler.CodebaseBranchHandler {
	if strings.ToLower(ciType) == util.GitlabCi {
		return createGitlabCiDefChain(client)
	}
	return createJenkinsDefChain(client)
}

func nextServeOrNil(next handler.CodebaseBranchHandler, cb *v1alpha1.CodebaseBranch) error {
	if next != nil {
		return next.ServeRequest(cb)
	}
	log.Info("handling of codebase branch has been finished", "name", cb.Name)
	return nil
}
