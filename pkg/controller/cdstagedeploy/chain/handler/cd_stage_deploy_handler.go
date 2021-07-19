package handler

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type CDStageDeployHandler interface {
	ServeRequest(stageDeploy *v1alpha1.CDStageDeploy) error
}
