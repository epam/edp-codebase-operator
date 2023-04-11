package argocd

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	applicationKind       = "Application"
	applicationAPIVersion = "v1alpha1"
	applicationGroup      = "argoproj.io"
)

type applicationOption struct {
	name      string
	namespace string
	spec      map[string]interface{}
	labels    map[string]string
}

// Option is a function that applies an option to applicationOption.
type Option func(*applicationOption)

// WithName sets the name of the application.
func WithName(name string) Option {
	return func(o *applicationOption) { o.name = name }
}

// WithNamespace sets the namespace of the application.
func WithNamespace(namespace string) Option {
	return func(o *applicationOption) { o.namespace = namespace }
}

// WithLabels sets the labels of the application.
func WithLabels(labels map[string]string) Option {
	return func(o *applicationOption) { o.labels = labels }
}

// WithSpec sets the spec of the application.
func WithSpec(spec map[string]interface{}) Option {
	return func(o *applicationOption) { o.spec = spec }
}

// NewArgoCDApplication returns a new ArgoCD Application unstructured object.
func NewArgoCDApplication(opts ...Option) *unstructured.Unstructured {
	option := &applicationOption{}
	for _, applyOption := range opts {
		applyOption(option)
	}

	application := &unstructured.Unstructured{}
	application.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   applicationGroup,
		Version: applicationAPIVersion,
		Kind:    applicationKind,
	})
	application.SetNamespace(option.namespace)
	application.SetName(option.name)
	application.SetLabels(option.labels)

	if option.spec != nil {
		application.Object["spec"] = option.spec
	}

	return application
}

// NewArgoCDApplicationList returns a new ArgoCD ApplicationList unstructured object.
func NewArgoCDApplicationList() *unstructured.UnstructuredList {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   applicationGroup,
		Version: applicationAPIVersion,
		Kind:    applicationKind,
	})

	return list
}
