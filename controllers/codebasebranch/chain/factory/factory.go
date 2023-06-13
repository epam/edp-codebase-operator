package factory

import (
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/clean_tmp_directory"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/empty"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_branch_in_git"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/trigger_job"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/update_perf_data_sources"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("codebase_branch_factory")

func createJenkinsDefChain(c client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "jenkins chain")

	return chain.CheckCommitHashExists{
		Client: c,
		Git:    &git.GitProvider{},
		Log:    ctrl.Log.WithName("check_commit_hash_exists"),
		Next: trigger_job.TriggerReleaseJob{
			TriggerJob: trigger_job.TriggerJob{
				Client: c,
				Service: &service.CodebaseBranchServiceProvider{
					Client: c,
				},
				Next: update_perf_data_sources.UpdatePerfDataSources{
					Client: c,
					Next: put_codebase_image_stream.PutCodebaseImageStream{
						Client: c,
						Next:   &clean_tmp_directory.CleanTempDirectory{},
					},
				},
			},
		},
	}
}

func createTektonDefChain(c client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "tekton chain")

	return put_branch_in_git.PutBranchInGit{
		Client: c,
		Git:    &git.GitProvider{},
		Next: update_perf_data_sources.UpdatePerfDataSources{
			Next: put_codebase_image_stream.PutCodebaseImageStream{
				Client: c,
				Next:   &clean_tmp_directory.CleanTempDirectory{},
			},
			Client: c,
		},
		Service: &service.CodebaseBranchServiceProvider{
			Client: c,
		},
	}
}

func GetDeletionChain(ciType string, c client.Client) handler.CodebaseBranchHandler {
	if strings.EqualFold(ciType, util.CITekton) {
		return empty.MakeChain("no deletion chain for tekton", false)
	}

	return trigger_job.TriggerDeletionJob{
		TriggerJob: trigger_job.TriggerJob{
			Client: c,
			Service: &service.CodebaseBranchServiceProvider{
				Client: c,
			},
		},
	}
}

func GetChain(ciType string, c client.Client) handler.CodebaseBranchHandler {
	if strings.EqualFold(ciType, util.CITekton) {
		return createTektonDefChain(c)
	}

	return createJenkinsDefChain(c)
}
