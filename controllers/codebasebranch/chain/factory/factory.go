package factory

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/clean_tmp_directory"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/empty"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_branch_in_git"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
)

var log = ctrl.Log.WithName("codebase_branch_factory")

func GetDeletionChain() handler.CodebaseBranchHandler {
	return empty.MakeChain("no deletion chain for tekton", false)
}

func GetChain(c client.Client) handler.CodebaseBranchHandler {
	log.Info("chain is selected", "type", "tekton chain")

	return put_branch_in_git.PutBranchInGit{
		Client: c,
		Git:    &git.GitProvider{},
		Next: put_codebase_image_stream.PutCodebaseImageStream{
			Client: c,
			Next:   &clean_tmp_directory.CleanTempDirectory{},
		},
		Service: &service.CodebaseBranchServiceProvider{
			Client: c,
		},
	}
}
