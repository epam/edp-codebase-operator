package factory

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/clean_tmp_directory"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/empty"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_branch_in_git"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
)

func GetDeletionChain() handler.CodebaseBranchHandler {
	return empty.MakeChain("no deletion chain for tekton", false)
}

func GetChain(c client.Client) handler.CodebaseBranchHandler {
	return put_branch_in_git.PutBranchInGit{
		Client: c,
		Git:    &git.GitProvider{},
		Next: chain.ProcessNewVersion{
			Client: c,
			Next: put_codebase_image_stream.PutCodebaseImageStream{
				Client: c,
				Next:   &clean_tmp_directory.CleanTempDirectory{},
			},
		},
		Service: &service.CodebaseBranchServiceProvider{
			Client: c,
		},
	}
}
