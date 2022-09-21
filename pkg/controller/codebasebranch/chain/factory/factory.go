package factory

import (
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/clean_tmp_directory"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/empty"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/put_branch_in_git"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/trigger_job"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/update_perf_data_sources"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("codebase_branch_factory")

func createJenkinsDefChain(client client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "jenkins chain")
	return trigger_job.TriggerReleaseJob{
		TriggerJob: trigger_job.TriggerJob{
			Client: client,
			Service: &service.CodebaseBranchServiceProvider{
				Client: client,
			},
			Next: update_perf_data_sources.UpdatePerfDataSources{
				Client: client,
				Next: put_codebase_image_stream.PutCodebaseImageStream{
					Client: client,
					Next:   clean_tmp_directory.CleanTempDirectory{},
				},
			},
		},
	}
}

func createGitlabCiDefChain(client client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "gitlab ci chain")
	return put_branch_in_git.PutBranchInGit{
		Client: client,
		Git:    &gitserver.GitProvider{},
		Next: update_perf_data_sources.UpdatePerfDataSources{
			Next: put_codebase_image_stream.PutCodebaseImageStream{
				Client: client,
				Next:   clean_tmp_directory.CleanTempDirectory{},
			},
			Client: client,
		},
		Service: &service.CodebaseBranchServiceProvider{
			Client: client,
		},
	}
}

func GetDeletionChain(ciType string, client client.Client) handler.CodebaseBranchHandler {
	if strings.EqualFold(ciType, util.GitlabCi) {
		return empty.MakeChain("no deletion chain for gitlab ci", false)
	}

	return trigger_job.TriggerDeletionJob{
		TriggerJob: trigger_job.TriggerJob{
			Client: client,
			Service: &service.CodebaseBranchServiceProvider{
				Client: client,
			},
		},
	}
}

func GetChain(ciType string, client client.Client) handler.CodebaseBranchHandler {
	if strings.EqualFold(ciType, util.GitlabCi) {
		return createGitlabCiDefChain(client)
	}
	return createJenkinsDefChain(client)
}
