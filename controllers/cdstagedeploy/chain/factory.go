package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/autodeploy"
	"github.com/epam/edp-codebase-operator/v2/pkg/tektoncd"
)

type CDStageDeployChain func(cl client.Client, stageDeploy *codebaseApi.CDStageDeploy) CDStageDeployHandler

func CreateChain(cl client.Client, _ *codebaseApi.CDStageDeploy) CDStageDeployHandler {
	c := chain{}

	c.Use(
		NewResolveStatus(cl),
		NewProcessTriggerTemplate(cl, tektoncd.NewTektonTriggerTemplateManager(cl), autodeploy.NewStrategyManager(cl)),
		NewDeleteCDStageDeploy(cl),
	)

	return &c
}
