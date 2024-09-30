package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/autodeploy"
	"github.com/epam/edp-codebase-operator/v2/pkg/tektoncd"
)

type CDStageDeployChain func(cl client.Client, stageDeploy *codebaseApi.CDStageDeploy) CDStageDeployHandler

func CreateChain(cl client.Client, stageDeploy *codebaseApi.CDStageDeploy) CDStageDeployHandler {
	c := chain{}

	if stageDeploy.Spec.TriggerType == pipelineApi.TriggerTypeAutoStable {
		c.Use(
			NewCreatePendingTriggerTemplate(cl, tektoncd.NewTektonTriggerTemplateManager(cl), autodeploy.NewStrategyManager(cl)),
			NewProcessPendingPipeRuns(cl),
			NewDeleteCDStageDeploy(cl),
		)

		return &c
	}

	c.Use(
		NewResolveStatus(cl),
		NewProcessTriggerTemplate(cl, tektoncd.NewTektonTriggerTemplateManager(cl), autodeploy.NewStrategyManager(cl)),
		NewDeleteCDStageDeploy(cl),
	)

	return &c
}
