package factory

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/clean_tmp_directory"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/put_branch_in_git"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/trigger_release_job"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/update_perf_data_sources"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("codebase_branch_factory")

func createJenkinsDefChain(client client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "jenkins chain")
	return trigger_release_job.TriggerReleaseJob{
		Client: client,
		Service: service.CodebaseBranchService{
			Client: client,
		},
		Next: update_perf_data_sources.UpdatePerfDataSources{
			Client: client,
		},
	}
}

func createGitlabCiDefChain(client client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "gitlab ci chain")
	return put_branch_in_git.PutBranchInGit{
		Client: client,
		Git:    gitserver.GitProvider{},
		Next: update_perf_data_sources.UpdatePerfDataSources{
			Next: put_codebase_image_stream.PutCodebaseImageStream{
				Client: client,
				Next:   clean_tmp_directory.CleanTempDirectory{},
			},
			Client: client,
		},
		Service: service.CodebaseBranchService{
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
