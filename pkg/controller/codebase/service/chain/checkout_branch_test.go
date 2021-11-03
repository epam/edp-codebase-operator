package chain

import (
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetRepositoryCredentialsIfExists_ShouldPass(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()
	u, p, err := GetRepositoryCredentialsIfExists(c, fakeCl)
	assert.Equal(t, u, util.GetStringP("user"))
	assert.Equal(t, p, util.GetStringP("pass"))
	assert.NoError(t, err)
}

func TestGetRepositoryCredentialsIfExists_ShouldFail(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	_, _, err := GetRepositoryCredentialsIfExists(c, fakeCl)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "Unable to get secret repository-codebase-fake-name-temp") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
