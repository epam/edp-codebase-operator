package template

import (
	"io/ioutil"
	"os"
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

const (
	fakeName      string = "fake-name"
	fakeNamespace string = "fake-namespace"
)

func TestPrepareTemplates_ShouldPass(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Type:             util.Application,
			Strategy:         v1alpha1.Create,
			DeploymentScript: "helm-chart",
			Lang:             "go",
		},
		Status: v1alpha1.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}

	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	err = PrepareTemplates(fakeCl, *c, dir, "../../../../../build")
	assert.NoError(t, err)
}

func TestPrepareTemplates_ShouldFailOnGetProjectUrl(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Type:             util.Application,
			Strategy:         "fake",
			DeploymentScript: "helm-chart",
			Lang:             "go",
		},
	}

	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	err := PrepareTemplates(fakeCl, *c, "/tmp", "../../../../../build")
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "unable get project url") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPrepareGitLabCITemplates(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Type:             util.Application,
			Strategy:         v1alpha1.Clone,
			DeploymentScript: "helm-chart",
			Lang:             "java",
			Repository: &v1alpha1.Repository{
				Url: "http://example.com",
			},
			Framework: util.GetStringP("java11"),
		},
		Status: v1alpha1.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}

	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	err = PrepareGitlabCITemplates(fakeCl, *c, dir, "../../../../../build")
	assert.NoError(t, err)
}

func TestGetProjectUrl_ShouldPass(t *testing.T) {

	c := &v1alpha1.Codebase{
		Spec: v1alpha1.CodebaseSpec{
			Type:       util.Application,
			Strategy:   util.ImportStrategy,
			GitUrlPath: util.GetStringP("/fake/repo.git"),
			GitServer:  fakeName,
		},
	}
	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			GitHost: fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs).Build()

	url, err := getProjectUrl(fakeCl, c.Spec, fakeNamespace)
	assert.NoError(t, err)
	assert.Equal(t, url, "https://fake-name/fake/repo.git")
}

func TestGetProjectUrl_ShouldFailToGetGitserver(t *testing.T) {

	c := &v1alpha1.Codebase{
		Spec: v1alpha1.CodebaseSpec{
			Type:       util.Application,
			Strategy:   util.ImportStrategy,
			GitUrlPath: util.GetStringP("/fake/repo.git"),
			GitServer:  fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	url, err := getProjectUrl(fakeCl, c.Spec, fakeNamespace)
	assert.Error(t, err)
	assert.Empty(t, url)
	if !strings.Contains(err.Error(), "unable get git server") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetProjectUrl_ShouldFailWithUnsupportStrategy(t *testing.T) {

	c := &v1alpha1.Codebase{
		Spec: v1alpha1.CodebaseSpec{
			Type:     util.Application,
			Strategy: "fake",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	url, err := getProjectUrl(fakeCl, c.Spec, fakeNamespace)
	assert.Error(t, err)
	assert.Empty(t, url)
	if !strings.Contains(err.Error(), "unable get project url, caused by the unsupported strategy") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
