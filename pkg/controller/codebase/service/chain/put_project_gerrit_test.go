package chain

import (
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

	ppg := PutProjectGerrit{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err := ppg.ServeRequest(c)
	assert.NoError(t, err)
}

func TestPutProjectGerrit_ShouldFailToRunSSHCommand(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer func() {
		os.RemoveAll(dir)
		os.Clearenv()
	}()

	os.Setenv("WORKING_DIR", dir)

	pk, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		t.Error("Unable to generate test private key")
	}
	privkey_bytes := x509.MarshalPKCS1PrivateKey(pk)
	privkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkey_bytes,
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
			util.PrivateSShKeyName: []byte(privkey_pem),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, s).Build()

	os.Setenv("ASSETS_DIR", "../../../../../build")
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

	ppg := PutProjectGerrit{
		client: fakeCl,
		git:    mGit,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to run ssh command")
}

func TestPutProjectGerrit_ShouldFailToGetConfgimap(t *testing.T) {
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

	ppg := PutProjectGerrit{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err := ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "couldn't get edp name: configmaps \"edp-config\" not found")
}

func TestPutProjectGerrit_ShouldFailToCreateRepo(t *testing.T) {
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

	err := ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mkdir /home/codebase-operator:")
}

func TestPutProjectGerrit_ShouldFailToGetGerritPort(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer func() {
		os.RemoveAll(dir)
		os.Clearenv()
	}()

	os.Setenv("WORKING_DIR", dir)

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

	err = ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable get gerrit port")
}

func TestPutProjectGerrit_ShouldFailToGetUserSettings(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer func() {
		os.RemoveAll(dir)
		os.Clearenv()
	}()

	os.Setenv("WORKING_DIR", dir)

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

	ppg := PutProjectGerrit{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable get user settings settings")
}

func TestPutProjectGerrit_ShouldFailToSetVCSIntegration(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer func() {
		os.RemoveAll(dir)
		os.Clearenv()
	}()

	os.Setenv("WORKING_DIR", dir)

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

	ppg := PutProjectGerrit{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create project in VCS")
}

func TestPutProjectGerrit_ShouldFailedOnInitialProjectProvisioning(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer func() {
		os.RemoveAll(dir)
		os.Clearenv()
	}()

	os.Setenv("WORKING_DIR", dir)

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

	os.Setenv("ASSETS_DIR", "../../../../../build")

	ppg := PutProjectGerrit{
		client: fakeCl,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = ppg.ServeRequest(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial provisioning of codebase fake-name has been failed")
}

func TestPutProjectGerrit_pushToGerrit_ShouldPass(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatal("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	if _, err = git.PlainInit(dir, false); err != nil {
		t.Error("Unable to create test git repo")
	}

	mGit := new(mockGit.MockGit)
	mGit.On("PushChanges", "idrsa", "project-creator", dir).Return(nil)

	ppg := PutProjectGerrit{
		git: mGit,
	}

	err = ppg.pushToGerrit(22, "idrsa", "fake-host", "c-name", dir, codebaseApi.Clone)
	assert.NoError(t, err)
}

func TestPutProjectGerrit_pushToGerrit_ShouldFailToAddRemoteLink(t *testing.T) {

	ppg := PutProjectGerrit{}
	err := ppg.pushToGerrit(22, "idrsa", "fake-host", "c-name", "/tmp", codebaseApi.Clone)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "couldn't add remote link to Gerrit")
}

func TestPutProjectGerrit_pushToGerrit_ShouldFailOnPush(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatal("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	if _, err = git.PlainInit(dir, false); err != nil {
		t.Error("Unable to create test git repo")
	}

	mGit := new(mockGit.MockGit)
	mGit.On("PushChanges", "idrsa", "project-creator", dir).Return(errors.New("FATAL: PUSH"))

	ppg := PutProjectGerrit{
		git: mGit,
	}

	err = ppg.pushToGerrit(22, "idrsa", "fake-host", "c-name", dir, codebaseApi.Clone)
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

	ppg := PutProjectGerrit{
		client: fakeCl,
		git:    mGit,
	}

	err := ppg.initialProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.NoError(t, err)
}

func TestPutProjectGerrit_emptyProjectProvisioningShouldFailOnInit(t *testing.T) {
	mGit := new(mockGit.MockGit)
	mGit.On("Init", "/tmp").Return(errors.New("FATAL:FAIL"))

	ppg := PutProjectGerrit{
		git: mGit,
	}

	err := ppg.emptyProjectProvisioning("/tmp", "c-name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FATAL:FAIL")
}

func TestPutProjectGerrit_emptyProjectProvisioningShouldFailOnCommit(t *testing.T) {
	mGit := new(mockGit.MockGit)
	mGit.On("Init", "/tmp").Return(nil)
	mGit.On("CommitChanges", "/tmp", "Initial commit").Return(errors.New("FATAL:FAIL"))

	ppg := PutProjectGerrit{
		git: mGit,
	}

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

	ppg := PutProjectGerrit{
		client: fakeCl,
	}

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

	ppg := PutProjectGerrit{
		client: fakeCl,
	}

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

	ppg := PutProjectGerrit{
		client: fakeCl,
		git:    mGit,
	}

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

	ppg := PutProjectGerrit{
		client: fakeCl,
		git:    mGit,
	}

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

	ppg := PutProjectGerrit{
		client: fakeCl,
		git:    mGit,
	}

	err := ppg.notEmptyProjectProvisioning(c, logr.DiscardLogger{}, "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "squash commits in a template repo has been failed")
}

func TestPutProjectGerrit_tryToSquashCommitsShouldReturnNil(t *testing.T) {
	ppg := PutProjectGerrit{}
	if err := ppg.tryToSquashCommits("workDir", "codebaseName", codebaseApi.Clone); err != nil {
		t.Fatal("Must not fail")
	}
}

func TestPutProjectGerrit_tryToSquashCommitsShouldFailOnCommitChanges(t *testing.T) {
	mGit := new(mockGit.MockGit)
	mGit.On("Init", "workDir").Return(nil)
	mGit.On("CommitChanges", "workDir", "Initial commit").Return(errors.New("FATAL: FAIL"))

	ppg := PutProjectGerrit{
		git: mGit,
	}
	err := ppg.tryToSquashCommits("workDir", "codebaseName", codebaseApi.Create)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "an error has occurred while committing all default content")
}

func TestPutProjectGerrit_tryToCloneShouldPassWithExistingRepo(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	if err != nil {
		t.Fatal("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	var (
		u = "user"
		p = "pass"
	)
	err = os.MkdirAll(fmt.Sprintf("%v/.git", dir), 0775)
	if err != nil {
		t.Fatal("unable to create .git directory for test")
	}

	ppg := PutProjectGerrit{}
	err = ppg.tryToCloneRepo("repourl", &u, &p, dir, "c-name")
	assert.NoError(t, err)
}

func TestReplaceDefaultBranch(t *testing.T) {
	mGit := mockGit.MockGit{}
	ppg := PutProjectGerrit{
		git: &mGit,
	}

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
