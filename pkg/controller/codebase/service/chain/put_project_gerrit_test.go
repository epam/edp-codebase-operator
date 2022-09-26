package chain

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	mockGit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutProjectGerrit_ShouldPassForPushedTemplate(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
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
			"edp_name": "edp-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err := ppg.ServeRequest(ctx, c)
	assert.NoError(t, err)
}

func TestPutProjectGerrit_ShouldFailToRunSSHCommand(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)

		os.Clearenv()
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	pk, err := rsa.GenerateKey(rand.Reader, 128)
	require.NoError(t, err)

	privkeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)

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
			Git: *util.GetStringP("some-status"),
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
			"vcs_integration_enabled":  "false",
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
			util.PrivateSShKeyName: privkeyPem,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, s).Build()

	err = os.Setenv("ASSETS_DIR", "../../../../../build")
	require.NoError(t, err)

	var (
		u = "user"
		p = "pass"
	)
	wd := util.GetWorkDir(fakeName, fakeNamespace)

	mGit := new(mockGit.MockGit)
	mGit.On("CheckPermissions", "https://github.com/epmd-edp/go--.git", &u, &p).Return(true)
	mGit.On("CloneRepository", "https://github.com/epmd-edp/go--.git",
		&u, &p, wd).Return(nil)
	mGit.On("Init", wd).Return(nil)
	mGit.On("CommitChanges", wd, "Initial commit").Return(nil)

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		mGit,
	)

	err = ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to run ssh command")
}

func TestPutProjectGerrit_ShouldFailToGetConfigmap(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	cm := &coreV1.ConfigMap{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err := ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "couldn't get edp name: configmaps \"edp-config\" not found")
}

func TestPutProjectGerrit_ShouldFailToCreateRepo(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"edp_name": "edp-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	ppg := PutProjectGerrit{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err := ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mkdir /home/codebase-operator:")
}

func TestPutProjectGerrit_ShouldFailToGetGerritPort(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)

		os.Clearenv()
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"edp_name": "edp-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err = ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable get gerrit port")
}

func TestPutProjectGerrit_ShouldFailToGetUserSettings(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)

		os.Clearenv()
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"edp_name":                "edp-name",
			"vcs_integration_enabled": "fake",
		},
	}
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm, gs).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err = ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable get user settings settings")
}

func TestPutProjectGerrit_ShouldFailToSetVCSIntegration(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)

		os.Clearenv()
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
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
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm, gs).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err = ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create project in VCS")
}

func TestPutProjectGerrit_ShouldFailedOnInitialProjectProvisioning(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)

		os.Clearenv()
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
			Git: *util.GetStringP("some-status"),
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
			"vcs_integration_enabled":  "false",
			"perf_integration_enabled": "true",
			"dns_wildcard":             "dns",
			"edp_name":                 "edp-name",
			"edp_version":              "2.2.2",
			"vcs_group_name_url":       "edp",
			"vcs_ssh_port":             "22",
			"vcs_tool_name":            "stub",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, cm).Build()

	err = os.Setenv("ASSETS_DIR", "../../../../../build")
	require.NoError(t, err)

	ppg := NewPutProjectGerrit(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		nil,
	)

	err = ppg.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial provisioning of codebase fake-name has been failed")
}

func TestPutProjectGerrit_pushToGerrit_ShouldPass(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	_, err = git.PlainInit(dir, false)
	require.NoError(t, err, "unable to create test git repo")

	mGit := new(mockGit.MockGit)
	mGit.On("PushChanges", "idrsa", "project-creator", dir).Return(nil)

	ppg := PutProjectGerrit{
		git: mGit,
	}

	err = ppg.pushToGerrit(22, "idrsa", "fake-host", "c-name", dir)
	assert.NoError(t, err)
}

func TestPutProjectGerrit_pushToGerrit_ShouldFailToAddRemoteLink(t *testing.T) {
	ppg := NewPutProjectGerrit(nil, nil, nil)

	err := ppg.pushToGerrit(22, "idrsa", "fake-host", "c-name", "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "couldn't add remote link to Gerrit")
}

func TestPutProjectGerrit_pushToGerrit_ShouldFailOnPush(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	_, err = git.PlainInit(dir, false)
	require.NoError(t, err, "unable to create test git repo")

	mGit := new(mockGit.MockGit)
	mGit.On("PushChanges", "idrsa", "project-creator", dir).Return(errors.New("FATAL: PUSH"))

	ppg := NewPutProjectGerrit(
		nil,
		nil,
		mGit,
	)

	err = ppg.pushToGerrit(22, "idrsa", "fake-host", "c-name", dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FATAL: PUSH")
}

func TestPutProjectGerrit_initialProjectProvisioningForEmptyProjectShouldPass(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			EmptyProject: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	mGit := new(mockGit.MockGit)
	mGit.On("Init", "/tmp").Return(nil)
	mGit.On("CommitChanges", "/tmp", "Initial commit").Return(nil)

	ppg := NewPutProjectGerrit(
		fakeCl,
		nil,
		mGit,
	)

	err := ppg.initialProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.NoError(t, err)
}

func TestPutProjectGerrit_emptyProjectProvisioningShouldFailOnInit(t *testing.T) {
	mGit := new(mockGit.MockGit)
	mGit.On("Init", "/tmp").Return(errors.New("FATAL:FAIL"))

	ppg := NewPutProjectGerrit(
		nil,
		nil,
		mGit,
	)

	err := ppg.emptyProjectProvisioning("/tmp", "c-name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FATAL:FAIL")
}

func TestPutProjectGerrit_emptyProjectProvisioningShouldFailOnCommit(t *testing.T) {
	mGit := new(mockGit.MockGit)
	mGit.On("Init", "/tmp").Return(nil)
	mGit.On("CommitChanges", "/tmp", "Initial commit").Return(errors.New("FATAL:FAIL"))

	ppg := NewPutProjectGerrit(
		nil,
		nil,
		mGit,
	)

	err := ppg.emptyProjectProvisioning("/tmp", "c-name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FATAL:FAIL")
}

func TestPutProjectGerrit_notEmptyProjectProvisioningShouldFailOnGetRepoUrl(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Clone,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		nil,
		nil,
	)

	err := ppg.notEmptyProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "couldn't build repo url")
}

func TestPutProjectGerrit_notEmptyProjectProvisioningShouldFailOnGetRepoCreds(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Clone,
			Repository: &codebaseApi.Repository{
				Url: "link",
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	ppg := NewPutProjectGerrit(
		fakeCl,
		nil,
		nil,
	)

	err := ppg.notEmptyProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to get repository credentials")
}

func TestPutProjectGerrit_notEmptyProjectProvisioningShouldFailOnCheckPermission(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Clone,
			Repository: &codebaseApi.Repository{
				Url: "link",
			},
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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, s).Build()

	var (
		u = "user"
		p = "pass"
	)
	mGit := new(mockGit.MockGit)
	mGit.On("CheckPermissions", "link", &u, &p).Return(false)

	ppg := NewPutProjectGerrit(
		fakeCl,
		nil,
		mGit,
	)

	err := ppg.notEmptyProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot get access to the repository link")
}

func TestPutProjectGerrit_notEmptyProjectProvisioningShouldFailOnCloneRepo(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Clone,
			Repository: &codebaseApi.Repository{
				Url: "link",
			},
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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, s).Build()

	var (
		u = "user"
		p = "pass"
	)
	mGit := new(mockGit.MockGit)
	mGit.On("CheckPermissions", "link", &u, &p).Return(true)
	mGit.On("CloneRepository", "link",
		&u, &p, "/tmp").Return(errors.New("FATAL: FAIL"))

	ppg := NewPutProjectGerrit(
		fakeCl,
		nil,
		mGit,
	)

	err := ppg.notEmptyProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cloning template project has been failed: FATAL: FAIL")
}

func TestPutProjectGerrit_notEmptyProjectProvisioningShouldFailOnSquash(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Create,
			Repository: &codebaseApi.Repository{
				Url: "link",
			},
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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, s).Build()

	var (
		u = "user"
		p = "pass"
	)
	mGit := new(mockGit.MockGit)
	mGit.On("CheckPermissions", "https://github.com/epmd-edp/--.git", &u, &p).Return(true)
	mGit.On("CloneRepository", "https://github.com/epmd-edp/--.git",
		&u, &p, "/tmp").Return(nil)
	mGit.On("Init", "/tmp").Return(errors.New("FATAL: FAIL"))

	ppg := NewPutProjectGerrit(
		fakeCl,
		nil,
		mGit,
	)

	err := ppg.notEmptyProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "squash commits in a template repo has been failed")
}

func TestPutProjectGerrit_tryToSquashCommitsShouldReturnNil(t *testing.T) {
	ppg := NewPutProjectGerrit(nil, nil, nil)
	err := ppg.tryToSquashCommits("workDir", "codebaseName", codebaseApi.Clone)
	assert.NoError(t, err)
}

func TestPutProjectGerrit_tryToSquashCommitsShouldFailOnCommitChanges(t *testing.T) {
	mGit := new(mockGit.MockGit)
	mGit.On("Init", "workDir").Return(nil)
	mGit.On("CommitChanges", "workDir", "Initial commit").Return(errors.New("FATAL: FAIL"))

	ppg := NewPutProjectGerrit(
		nil,
		nil,
		mGit,
	)
	err := ppg.tryToSquashCommits("workDir", "codebaseName", codebaseApi.Create)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "an error has occurred while committing all default content")
}

func TestPutProjectGerrit_tryToCloneShouldPassWithExistingRepo(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	_, err = git.PlainInit(dir, false)
	require.NoError(t, err, "unable to create test git repo")

	var (
		u = "user"
		p = "pass"
	)
	err = os.MkdirAll(fmt.Sprintf("%v/.git", dir), 0775)
	require.NoError(t, err, "unable to create .git directory for test")

	ppg := NewPutProjectGerrit(nil, nil, nil)
	err = ppg.tryToCloneRepo("repourl", &u, &p, dir, "c-name")
	assert.NoError(t, err)
}

func TestReplaceDefaultBranch(t *testing.T) {
	mGit := mockGit.MockGit{}
	ppg := NewPutProjectGerrit(
		nil,
		nil,
		&mGit,
	)

	mGit.On("RemoveBranch", "foo", "bar").Return(nil).Once()
	mGit.On("CreateChildBranch", "foo", "baz", "bar").Return(nil).Once()

	err := ppg.replaceDefaultBranch("foo", "bar", "baz")
	assert.NoError(t, err)

	mGit.On("RemoveBranch", "foo", "bar").Return(errors.New("RemoveBranch fatal")).Once()
	err = ppg.replaceDefaultBranch("foo", "bar", "baz")
	assert.EqualError(t, err, "unable to remove master branch: RemoveBranch fatal")

	mGit.On("RemoveBranch", "foo", "bar").Return(nil).Once()
	mGit.On("CreateChildBranch", "foo", "baz", "bar").Return(errors.New("fatal")).Once()

	err = ppg.replaceDefaultBranch("foo", "bar", "baz")
	assert.EqualError(t, err, "unable to create child branch: fatal")
}
