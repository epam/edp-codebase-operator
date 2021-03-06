package handler

import (
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type CodebaseBranchHandler interface {
	ServeRequest(c *edpv1alpha1.CodebaseBranch) error
}

var log = ctrl.Log.WithName("codebase_branch_handler")

func NextServeOrNil(next CodebaseBranchHandler, cb *edpv1alpha1.CodebaseBranch) error {
	if next != nil {
		return next.ServeRequest(cb)
	}
	log.Info("handling of codebase branch has been finished", "name", cb.Name)
	return nil
}
