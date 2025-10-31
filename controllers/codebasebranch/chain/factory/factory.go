package factory

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/clean_tmp_directory"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/empty"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_branch_in_git"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/put_codebase_image_stream"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func GetDeletionChain() handler.CodebaseBranchHandler {
	return empty.MakeChain("no deletion chain for tekton", false)
}

// DefaultGitProviderFactory creates a v2.GitProvider with credentials from GitServer and Secret.
func DefaultGitProviderFactory(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
	config := gitproviderv2.Config{
		SSHKey:      string(secret.Data[util.PrivateSShKeyName]),
		SSHUser:     gitServer.Spec.GitUser,
		SSHPort:     gitServer.Spec.SshPort,
		GitProvider: gitServer.Spec.GitProvider,
		Token:       string(secret.Data[util.GitServerSecretTokenField]),
	}

	return gitproviderv2.NewGitProvider(config)
}

func GetChain(c client.Client) handler.CodebaseBranchHandler {
	return chain.CheckReferenceExists{
		Client:             c,
		GitProviderFactory: DefaultGitProviderFactory,
		Next: put_branch_in_git.PutBranchInGit{
			Client:             c,
			GitProviderFactory: DefaultGitProviderFactory,
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
		},
	}
}
