package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CDStageDeployChain func(cl client.Client) CDStageDeployHandler

func CreateDefChain(cl client.Client) CDStageDeployHandler {
	c := chain{}

	c.Use(
		NewResolveStatus(cl),
		NewProcessTriggerTemplate(cl),
		NewDeleteCDStageDeploy(cl),
	)

	return &c
}
