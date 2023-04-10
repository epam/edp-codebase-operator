package chain

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestCleaner_ShouldPass(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "failed to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	t.Setenv("WORKING_DIR", dir)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, ssh).Build()

	cl := NewCleaner(fakeCl)

	err = cl.ServeRequest(ctx, c)
	assert.NoError(t, err)
}

func TestCleaner_ShouldNotFailedIfSecretNotFound(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "failed to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	t.Setenv("WORKING_DIR", dir)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	ssh := &coreV1.Secret{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, ssh).Build()

	cl := NewCleaner(fakeCl)

	err = cl.ServeRequest(ctx, c)
	assert.NoError(t, err)
}

func TestCleaner_ShouldFail(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "failed to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	t.Setenv("WORKING_DIR", dir)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	cl := NewCleaner(fakeCl)

	err = cl.ServeRequest(ctx, c)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to delete secret repository-codebase-fake-name-temp")
}
