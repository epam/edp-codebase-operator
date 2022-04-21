package chain

import (
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//TODO: remove global variable log
var log = ctrl.Log.WithName("codebase_handler")

func CreateGerritDefChain(client client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gerrit")
	gp := gitserver.GitProvider{}
	return PutProjectGerrit{
		next: PutGerritReplication{
			next: PutPerfDataSources{
				next: PutDeployConfigs{
					next: PutVersionFile{
						next: PutJenkinsFolder{
							next: Cleaner{
								client: client,
							},
							client: client,
						},
						client: client,
						cr:     cr,
						git:    gp,
					},
					client: client,
					cr:     cr,
					git:    gp,
				},
				client: client,
			},
			client: client,
		},
		client: client,
		cr:     cr,
		git:    gp,
	}
}

func CreateThirdPartyVcsProviderDefChain(client client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "third party VCS provider")
	gp := gitserver.GitProvider{}
	return CloneGitProject{
		next: PutPerfDataSources{
			next: PutDeployConfigsToGitProvider{
				next: PutVersionFile{
					next: PutJenkinsFolder{
						next: Cleaner{
							client: client,
						},
						client: client,
					},
					client: client,
					cr:     cr,
					git:    gp,
				},
				client: client,
				cr:     cr,
				git:    gp,
			},
			client: client,
		},
		git:    gp,
		client: client,
	}
}

func CreateDeletionChain(k8sClient client.Client) handler.CodebaseHandler {
	return DropJenkinsFolders{
		k8sClient: k8sClient,
	}
}

func CreateGitlabCiDefChain(client client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gitlab ci")
	gp := gitserver.GitProvider{}
	return CloneGitProject{
		next: PutPerfDataSources{
			next: PutGitlabCiDeployConfigs{
				next: PutGitlabCiFile{
					next: PutVersionFile{
						next: Cleaner{
							client: client,
						},
						client: client,
						cr:     cr,
						git:    gp,
					},
					client: client,
					cr:     cr,
					git:    gp,
				},
				client: client,
				cr:     cr,
				git:    gp,
			},
			client: client,
		},
		git:    gp,
		client: client,
	}
}

func nextServeOrNil(next handler.CodebaseHandler, codebase *edpv1alpha1.Codebase) error {
	if next != nil {
		return next.ServeRequest(codebase)
	}
	log.Info("handling of codebase has been finished", "codebase_name", codebase.Name)
	return nil
}
