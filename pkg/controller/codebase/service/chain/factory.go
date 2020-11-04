package chain

import (
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("codebase_handler")

func CreateGerritDefChain(cs openshift.ClientSet, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gerrit")
	gp := gitserver.GitProvider{}
	return PutProjectGerrit{
		next: PutGerritReplication{
			next: PutPerfDataSources{
				next: PutDeployConfigs{
					next: PutVersionFile{
						next: PutJenkinsFolder{
							next: Cleaner{
								clientSet: cs,
							},
							clientSet: cs,
						},
						clientSet: cs,
						cr:        cr,
						git:       gp,
					},
					clientSet: cs,
					cr:        cr,
					git:       gp,
				},
				client: cs.Client,
			},
			clientSet: cs,
		},
		clientSet: cs,
		cr:        cr,
		git:       gp,
	}
}

func CreateThirdPartyVcsProviderDefChain(cs openshift.ClientSet, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "third party VCS provider")
	gp := gitserver.GitProvider{}
	return CloneGitProject{
		next: PutPerfDataSources{
			next: PutDeployConfigsToGitProvider{
				next: PutVersionFile{
					next: PutJenkinsFolder{
						next: Cleaner{
							clientSet: cs,
						},
						clientSet: cs,
					},
					clientSet: cs,
					cr:        cr,
					git:       gp,
				},
				clientSet: cs,
				cr:        cr,
				git:       gp,
			},
			client: cs.Client,
		},
		clientSet: cs,
		git:       gp,
	}
}

func CreateGitlabCiDefChain(cs openshift.ClientSet, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gitlab ci")
	gp := gitserver.GitProvider{}
	return CloneGitProject{
		next: PutPerfDataSources{
			next: PutGitlabCiDeployConfigs{
				next: PutGitlabCiFile{
					next: PutVersionFile{
						next: Cleaner{
							clientSet: cs,
						},
						clientSet: cs,
						cr:        cr,
						git:       gp,
					},
					client: cs.Client,
					cr:     cr,
					git:    gp,
				},
				clientSet: cs,
				cr:        cr,
				git:       gp,
			},
			client: cs.Client,
		},
		clientSet: cs,
		git:       gp,
	}
}

func nextServeOrNil(next handler.CodebaseHandler, codebase *edpv1alpha1.Codebase) error {
	if next != nil {
		return next.ServeRequest(codebase)
	}
	log.Info("handling of codebase has been finished", "codebase name", codebase.Name)
	return nil
}
