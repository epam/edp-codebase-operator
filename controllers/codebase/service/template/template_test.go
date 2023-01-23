package template

import (
	"os"
	"strings"
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
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

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

	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"edp_name":                 "edp-name",
			"vcs_integration_enabled":  "false",
			"perf_integration_enabled": "false",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	err = PrepareTemplates(fakeCl, c, dir, "../../../../build")
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

	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"edp_name":                 "edp-name",
			"vcs_integration_enabled":  "false",
			"perf_integration_enabled": "false",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	err := PrepareTemplates(fakeCl, c, "/tmp", "../../../../build")
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "unable get project url") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPrepareGitLabCITemplates(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:             util.Application,
			Strategy:         codebaseApi.Clone,
			DeploymentScript: "helm-chart",
			Lang:             "java",
			Repository: &codebaseApi.Repository{
				Url: "http://example.com",
			},
			Framework: util.GetStringP("java11"),
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}

	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"edp_name":                 "edp-name",
			"vcs_integration_enabled":  "false",
			"perf_integration_enabled": "false",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	err = PrepareGitlabCITemplates(fakeCl, c, dir, "../../../../build")
	assert.NoError(t, err)
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

	if !strings.Contains(err.Error(), "unable get git server") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
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

	if !strings.Contains(err.Error(), "unable get project url, caused by the unsupported strategy") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}