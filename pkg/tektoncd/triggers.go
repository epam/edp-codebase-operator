package tektoncd

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	triggersGroup      = "triggers.tekton.dev"
	triggersAPIVersion = "v1beta1"
	triggersKind       = "EventListener"
)

func NewEventListenerUnstructured() *unstructured.Unstructured {
	application := &unstructured.Unstructured{}
	application.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   triggersGroup,
		Version: triggersAPIVersion,
		Kind:    triggersKind,
	})

	return application
}
