package chain

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	mockGit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-common/pkg/mock"
)

func TestPutDeployConfigs_ShouldPass(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:             util.Application,
			DeploymentScript: util.HelmChartDeploymentScriptType,
			Strategy:         codebaseApi.Create,
			Lang:             util.LanguageGo,
			DefaultBranch:    "fake-defaultBranch",
			GitUrlPath:       util.GetStringP(fakeName),
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			GitServer: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
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
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit-project-creator",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, s).Build()

	err = os.Setenv("ASSETS_DIR", "../../../../../build")
	require.NoError(t, err)

	port := int32(22)
	u := "user"
	p := "pass"
	wd := util.GetWorkDir(fakeName, fakeNamespace)

	mGit := new(mockGit.MockGit)
	mGit.On("CloneRepositoryBySsh", "fake",
		"project-creator", fmt.Sprintf("ssh://gerrit.%v:%v", fakeNamespace, fakeName),
		wd, port).Return(nil)

	mGit.On("CheckPermissions", "https://github.com/epmd-edp/go--.git", &u, &p).Return(true)
	mGit.On("GetCurrentBranchName", wd).Return("master", nil)
	mGit.On("Checkout", &u, &p, wd, "fake-defaultBranch", false).Return(nil)
	mGit.On("CommitChanges", wd, fmt.Sprintf("Add deployment templates for %v", c.Name)).Return(nil)
	mGit.On("PushChanges", "fake", "project-creator", wd, port).Return(nil)

	pdc := NewPutDeployConfigs(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		mGit,
	)

	err = pdc.ServeRequest(ctx, c)
	assert.NoError(t, err)
}

func TestPutDeployConfigs_ShouldFailOnGetGerritPort(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer: fakeName,
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
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
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit-project-creator",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm).Build()

	err = os.Setenv("ASSETS_DIR", "../../../../../build")
	require.NoError(t, err)

	pdc := NewPutDeployConfigs(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err = pdc.ServeRequest(ctx, c)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable get gerrit port", "wrong error returned")
}

func TestPutDeployConfigs_ServeRequest_Skip(t *testing.T) {
	ctx := context.Background()
	pdc := NewPutDeployConfigs(nil, nil, nil)
	logger := mock.Logger{}
	log = &logger
	cb := &codebaseApi.Codebase{
		Spec: codebaseApi.CodebaseSpec{DisablePutDeployTemplates: true},
	}
	expectedLog := "skip of putting deploy templates to codebase due to specified flag"

	err := pdc.ServeRequest(ctx, cb)
	assert.NoError(t, err)

	_, ok := logger.InfoMessages[expectedLog]
	assert.True(t, ok)

	delete(logger.InfoMessages, expectedLog)

	pdctp := PutDeployConfigsToGitProvider{}

	err = pdctp.ServeRequest(ctx, cb)
	assert.NoError(t, err)

	_, ok = logger.InfoMessages[expectedLog]
	assert.True(t, ok)

	delete(logger.InfoMessages, expectedLog)

	pdGitlab := PutGitlabCiDeployConfigs{}

	err = pdGitlab.ServeRequest(ctx, cb)
	assert.NoError(t, err)

	_, ok = logger.InfoMessages[expectedLog]
	assert.True(t, ok)
}
