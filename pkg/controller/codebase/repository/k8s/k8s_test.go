package k8s

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestK8SCodebaseRepository_SelectProjectStatusValue(t *testing.T) {
	client := fake.NewFakeClient()
	cr := v1alpha1.Codebase{
		Status: v1alpha1.CodebaseStatus{
			Git: "status-quo",
		},
	}
	repo := NewK8SCodebaseRepository(client, &cr)

	status, err := repo.SelectProjectStatusValue("ignore", "ignore")

	assert.Equal(t, "status-quo", *status)
	assert.NoError(t, err)
}

func TestK8SCodebaseRepository_UpdateProjectStatusValue(t *testing.T) {
	cr := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "super-edp",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &cr)
	objs := []runtime.Object{
		&cr,
	}
	client := fake.NewFakeClient(objs...)
	repo := NewK8SCodebaseRepository(client, &cr)

	err := repo.UpdateProjectStatusValue("status-quo", "ignore", "ignore")

	assert.NoError(t, err)
}

func TestK8SCodebaseRepository_UpdateProjectStatusValueNotFound(t *testing.T) {
	cr := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "super-edp",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &cr)
	client := fake.NewFakeClient()
	repo := NewK8SCodebaseRepository(client, &cr)

	err := repo.UpdateProjectStatusValue("status-quo", "ignore", "ignore")

	assert.Error(t, err)
}
