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

func createJenkinsDefChain(c client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "jenkins chain")
	return trigger_job.TriggerReleaseJob{
		TriggerJob: trigger_job.TriggerJob{
			Client: c,
			Service: &service.CodebaseBranchServiceProvider{
				Client: c,
			},
			Next: update_perf_data_sources.UpdatePerfDataSources{
				Client: c,
				Next: put_codebase_image_stream.PutCodebaseImageStream{
					Client: c,
					Next:   clean_tmp_directory.CleanTempDirectory{},
				},
			},
		},
	}
}

func createGitlabCiDefChain(c client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "gitlab ci chain")
	return put_branch_in_git.PutBranchInGit{
		Client: c,
		Git:    &gitserver.GitProvider{},
		Next: update_perf_data_sources.UpdatePerfDataSources{
			Next: put_codebase_image_stream.PutCodebaseImageStream{
				Client: c,
				Next:   clean_tmp_directory.CleanTempDirectory{},
			},
			Client: c,
		},
		Service: &service.CodebaseBranchServiceProvider{
			Client: c,
		},
	}
}

func createTektonDefChain(c client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "tekton chain")
	return put_branch_in_git.PutBranchInGit{
		Client: c,
		Git:    &gitserver.GitProvider{},
		Next: update_perf_data_sources.UpdatePerfDataSources{
			Next: put_codebase_image_stream.PutCodebaseImageStream{
				Client: c,
				Next:   clean_tmp_directory.CleanTempDirectory{},
			},
			Client: c,
		},
		Service: &service.CodebaseBranchServiceProvider{
			Client: c,
		},
	}
}

func GetDeletionChain(ciType string, c client.Client) handler.CodebaseBranchHandler {
	if strings.EqualFold(ciType, util.GitlabCi) {
		return empty.MakeChain("no deletion chain for gitlab ci", false)
	}

	if strings.EqualFold(ciType, util.Tekton) {
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
	if strings.EqualFold(ciType, util.GitlabCi) {
		return createGitlabCiDefChain(c)
	}
	if strings.EqualFold(ciType, util.Tekton) {
		return createTektonDefChain(c)
	}
	return createJenkinsDefChain(c)
}
