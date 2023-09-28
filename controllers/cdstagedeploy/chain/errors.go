package chain

import "errors"

var (
	// ErrMultipleArgoApplicationsFound is returned when multiple ArgoCD Applications are found.
	ErrMultipleArgoApplicationsFound = errors.New("multiple ArgoCD Applications found")
	// ErrArgoApplicationNotFound is returned when ArgoCD Application is not found.
	ErrArgoApplicationNotFound = errors.New("ArgoCD Application not found")
)
