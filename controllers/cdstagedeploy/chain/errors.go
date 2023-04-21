package chain

import "errors"

var (
	// ErrCDStageJenkinsDeploymentHasNotBeenProcessed is returned when Jenkins Deployment has not been processed.
	ErrCDStageJenkinsDeploymentHasNotBeenProcessed = errors.New("failed to process for previous version of application")
	// ErrMultipleArgoApplicationsFound is returned when multiple ArgoCD Applications are found.
	ErrMultipleArgoApplicationsFound = errors.New("multiple ArgoCD Applications found")
	// ErrArgoApplicationNotFound is returned when ArgoCD Application is not found.
	ErrArgoApplicationNotFound = errors.New("ArgoCD Application not found")
)
