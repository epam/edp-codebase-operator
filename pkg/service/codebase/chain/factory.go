package chain

import (
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/chain/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("codebase_handler")

func CreateGerritDefChain(cs openshift.ClientSet) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gerrit")
	return PutProjectGerrit{
		next: PutGerritReplication{
			next: SetupPerf{
				next: PutDeployConfigs{
					next: PutS2iIs{
						next: PutJenkinsFolder{
							next: Cleaner{
								clientSet: cs,
							},
							clientSet: cs,
						},
						clientSet: cs,
					},
					clientSet: cs,
				},
				clientSet: cs,
			},
			clientSet: cs,
		},
		clientSet: cs,
	}
}

func CreateThirdPartyVcsProviderDefChain(cs openshift.ClientSet) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "third party VCS")
	return CloneGitProject{
		next: PutDeployConfigsToGitProvider{
			next: PutS2iIs{
				next: PutJenkinsFolder{
					next: Cleaner{
						clientSet: cs,
					},
					clientSet: cs,
				},
				clientSet: cs,
			},
			clientSet: cs,
		},
		clientSet: cs,
	}
}

func nextServeOrNil(next handler.CodebaseHandler, codebase *edpv1alpha1.Codebase) error {
	if next != nil {
		return next.ServeRequest(codebase)
	}
	log.Info("handling of codebase has been finished", "codebase name", codebase.Name)
	return nil
}
