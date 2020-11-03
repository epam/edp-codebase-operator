package util

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetStringP(val string) *string {
	return &val
}

func GetWorkDir(codebaseName, namespace string) string {
	return fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v/%v", namespace, codebaseName, "templates", codebaseName)
}

func GetOwnerReference(ownerKind string, ors []metav1.OwnerReference) (*metav1.OwnerReference, error) {
	log.V(2).Info("finding owner", "kind", ownerKind)
	if len(ors) == 0 {
		return nil, fmt.Errorf("resource doesn't have %v owner reference", ownerKind)
	}
	for _, o := range ors {
		if o.Kind == ownerKind {
			return &o, nil
		}
	}
	return nil, fmt.Errorf("resource doesn't have %v owner reference", ownerKind)
}
