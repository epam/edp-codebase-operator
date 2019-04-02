package impl

import (
	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationBranchService struct {
	Client         client.Client
}

func (applicationBranch ApplicationBranchService) Create(cr *edpv1alpha1.ApplicationBranch) {

}

func (applicationBranch ApplicationBranchService) Update(cr *edpv1alpha1.ApplicationBranch) {

}

func (applicationBranch ApplicationBranchService) Delete(cr *edpv1alpha1.ApplicationBranch) {

}
