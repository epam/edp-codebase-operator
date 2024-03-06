package chain

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CDStageDeployChain func(cl client.Client) (CDStageDeployHandler, error)

func CreateDefChain(cl client.Client) (CDStageDeployHandler, error) {
	c := chain{}

	c.Use(
		NewProcessTriggerTemplate(cl),
		NewDeleteCDStageDeploy(cl),
	)

	return &c, nil
}
