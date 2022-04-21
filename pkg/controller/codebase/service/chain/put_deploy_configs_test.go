package chain

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	mockgit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-common/pkg/mock"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutDeployConfigs_ShouldPass(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	os.Setenv("WORKING_DIR", dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Type:             util.Application,
			DeploymentScript: util.HelmChartDeploymentScriptType,
			Strategy:         v1alpha1.Create,
			Lang:             util.LanguageGo,
			DefaultBranch:    "fake-defaultBranch",
			GitUrlPath:       util.GetStringP(fakeName),
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
			GitServer: fakeName,
		},
		Status: v1alpha1.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
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

	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"vcs_integration_enabled":  "true",
			"perf_integration_enabled": "true",
			"dns_wildcard":             "dns",
			"edp_name":                 "edp-name",
			"edp_version":              "2.2.2",
			"vcs_group_name_url":       "edp",
			"vcs_ssh_port":             "22",
			"vcs_tool_name":            "stub",
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit-project-creator",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, s).Build()

	os.Setenv("ASSETS_DIR", "../../../../../build")
	var (
		port int32 = 22
		u          = "user"
		p          = "pass"
	)
	wd := util.GetWorkDir(fakeName, fakeNamespace)

	mGit := new(mockgit.MockGit)
	mGit.On("CloneRepositoryBySsh", "fake",
		"project-creator", fmt.Sprintf("ssh://gerrit.%v:%v", fakeNamespace, fakeName),
		wd, port).Return(nil)

	mGit.On("CheckPermissions", "https://github.com/epmd-edp/go--.git", &u, &p).Return(true)
	mGit.On("GetCurrentBranchName", wd).Return("master", nil)
	mGit.On("Checkout", &u, &p, wd, "fake-defaultBranch", false).Return(nil)
	mGit.On("CommitChanges", wd, fmt.Sprintf("Add deployment templates for %v", c.Name)).Return(nil)
	mGit.On("PushChanges", "fake", "project-creator", wd).Return(nil)

	pdc := PutDeployConfigs{
		client: fakeCl,
		git:    mGit,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = pdc.ServeRequest(c)
	assert.NoError(t, err)
}

func TestPutDeployConfigs_ShouldFailOnGetGerritPort(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	os.Setenv("WORKING_DIR", dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer: fakeName,
		},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"vcs_integration_enabled":  "true",
			"perf_integration_enabled": "true",
			"dns_wildcard":             "dns",
			"edp_name":                 "edp-name",
			"edp_version":              "2.2.2",
			"vcs_group_name_url":       "edp",
			"vcs_ssh_port":             "22",
			"vcs_tool_name":            "stub",
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit-project-creator",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm).Build()

	os.Setenv("ASSETS_DIR", "../../../../../build")

	pdc := PutDeployConfigs{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = pdc.ServeRequest(c)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "unable get gerrit port") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutDeployConfigs_ServeRequest_Skip(t *testing.T) {
	pdc := PutDeployConfigs{}
	logger := mock.Logger{}
	log = &logger
	cb := &v1alpha1.Codebase{
		Spec: v1alpha1.CodebaseSpec{DisablePutDeployTemplates: true},
	}
	expectedLog := "skip of putting deploy templates to codebase due to specified flag"
	err := pdc.ServeRequest(cb)
	assert.NoError(t, err)
	_, ok := logger.InfoMessages[expectedLog]
	assert.True(t, ok)

	delete(logger.InfoMessages, expectedLog)

	pdctp := PutDeployConfigsToGitProvider{}
	err = pdctp.ServeRequest(cb)
	assert.NoError(t, err)
	_, ok = logger.InfoMessages[expectedLog]
	assert.True(t, ok)

	delete(logger.InfoMessages, expectedLog)

	pdGitlab := PutGitlabCiDeployConfigs{}
	err = pdGitlab.ServeRequest(cb)
	assert.NoError(t, err)
	_, ok = logger.InfoMessages[expectedLog]
	assert.True(t, ok)
}
