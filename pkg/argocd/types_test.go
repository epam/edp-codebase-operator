package argocd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewArgoCDApplication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []Option
		want *unstructured.Unstructured
	}{
		{
			name: "should create application with default values",
			opts: []Option{},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
				},
			},
		},
		{
			name: "should create application with name",
			opts: []Option{WithName("test")},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			},
		},
		{
			name: "should create application with namespace",
			opts: []Option{WithNamespace("default")},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata": map[string]interface{}{
						"namespace": "default",
					},
				},
			},
		},
		{
			name: "should create application with labels",
			opts: []Option{WithLabels(map[string]string{"test": "test"})},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
		},
		{
			name: "should create application with spec",
			opts: []Option{WithSpec(map[string]interface{}{"test": "test"})},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"spec": map[string]interface{}{
						"test": "test",
					},
				},
			},
		},
		{
			name: "should create application with all values",
			opts: []Option{
				WithName("test"),
				WithNamespace("default"),
				WithLabels(map[string]string{"test": "test"}),
				WithSpec(map[string]interface{}{"test": "test"}),
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata": map[string]interface{}{
						"name":      "test",
						"namespace": "default",
						"labels": map[string]interface{}{
							"test": "test",
						},
					},
					"spec": map[string]interface{}{
						"test": "test",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewArgoCDApplication(tt.opts...)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewArgoCDApplicationList(t *testing.T) {
	tests := []struct {
		name string
		want *unstructured.UnstructuredList
	}{
		{
			name: "should create application list",
			want: &unstructured.UnstructuredList{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewArgoCDApplicationList())
		})
	}
}
