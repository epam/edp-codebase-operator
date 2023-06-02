package template

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	fakeName      string = "fake-name"
	fakeNamespace string = "fake-namespace"
)

func TestPrepareTemplates_ShouldPass(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:             util.Application,
			Strategy:         codebaseApi.Create,
			DeploymentScript: "helm-chart",
			Lang:             "go",
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}
	config := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, coreV1.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, config).Build()

	t.Setenv(util.AssetsDirEnv, "../../../../build")

	err := PrepareTemplates(context.Background(), fakeCl, c, t.TempDir())
	assert.NoError(t, err)
}

func TestPrepareTemplates_ShouldSkipSonarConfig(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:             util.Application,
			Strategy:         codebaseApi.Create,
			DeploymentScript: "helm-chart",
			Lang:             "go",
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}
	config := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, coreV1.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, config).Build()

	t.Setenv(util.AssetsDirEnv, "../../../../build")

	wd := t.TempDir()
	_, err := os.Create(path.Join(wd, "sonar-project.properties"))
	require.NoError(t, err)

	err = PrepareTemplates(context.Background(), fakeCl, c, wd)
	assert.NoError(t, err)
}

func TestPrepareTemplates_ShouldFailOnGetProjectUrl(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:             util.Application,
			Strategy:         "fake",
			DeploymentScript: "helm-chart",
			Lang:             "go",
		},
	}
	config := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, config).Build()

	t.Setenv(util.AssetsDirEnv, "../../../../build")

	err := PrepareTemplates(context.Background(), fakeCl, c, "/tmp")
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "failed to get project url")
}

func TestGetProjectUrl_ShouldPass(t *testing.T) {
	c := &codebaseApi.Codebase{
		Spec: codebaseApi.CodebaseSpec{
			Type:       util.Application,
			Strategy:   util.ImportStrategy,
			GitUrlPath: util.GetStringP("/fake/repo.git"),
			GitServer:  fakeName,
		},
	}
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost: fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs).Build()

	url, err := getProjectUrl(fakeCl, &c.Spec, fakeNamespace)
	assert.NoError(t, err)
	assert.Equal(t, url, "https://fake-name/fake/repo.git")
}

func TestGetProjectUrl_ShouldFailToGetGitServer(t *testing.T) {
	c := &codebaseApi.Codebase{
		Spec: codebaseApi.CodebaseSpec{
			Type:       util.Application,
			Strategy:   util.ImportStrategy,
			GitUrlPath: util.GetStringP("/fake/repo.git"),
			GitServer:  fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	url, err := getProjectUrl(fakeCl, &c.Spec, fakeNamespace)
	assert.Error(t, err)
	assert.Empty(t, url)

	assert.Contains(t, err.Error(), "failed to get git server")
}

func TestGetProjectUrl_ShouldFailWithUnsupportedStrategy(t *testing.T) {
	c := &codebaseApi.Codebase{
		Spec: codebaseApi.CodebaseSpec{
			Type:     util.Application,
			Strategy: "fake",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	url, err := getProjectUrl(fakeCl, &c.Spec, fakeNamespace)
	assert.Error(t, err)
	assert.Empty(t, url)

	assert.Contains(t, err.Error(), "failed to get project url, caused by the unsupported strategy")
}
