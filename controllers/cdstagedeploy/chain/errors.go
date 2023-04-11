package chain

import "errors"

var (
	ErrCDStageJenkinsDeploymentHasNotBeenProcessed = errors.New("failed to process for previous version of application")
	ErrMultipleArgoApplicationsFound               = errors.New("multiple ArgoCD Applications found")
)
