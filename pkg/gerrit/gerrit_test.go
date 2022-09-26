package gerrit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupSuite(tb testing.TB) (func(tb testing.TB), string) {
	log.Println("setup suite")
	pk, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		tb.Error("Unable to generate test private key")
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	privateKeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		},
	)
	// Return a function to teardown the test
	return func(tb testing.TB) {
		log.Println("teardown suite")
	}, string(privateKeyPem)
}

func TestGenerateReplicationConfig(t *testing.T) {
	templatePath := "../../build/templates/gerrit"
	templateName := "replication-conf.tmpl"
	params := ReplicationConfigParams{
		Name:      "fake-name",
		VcsSshUrl: "ssh://repo.example.com",
	}

	p, err := generateReplicationConfig(templatePath, templateName, params)
	assert.NoError(t, err)
	assert.Contains(t, p, "[remote \"fake-name\"]\n  url = ssh://repo.example.com")
}

func TestSShInit_ShouldPass(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	g, err := SshInit(22, idrsa, "fake-host", logr.DiscardLogger{})
	assert.NoError(t, err)
	assert.Equal(t, g.Config.User, "project-creator")
	assert.Equal(t, g.Host, "fake-host")
	assert.Equal(t, g.Port, int32(22))
}

func TestSShInit_ShouldFailForIncorrectRSAPKey(t *testing.T) {
	gc, err := SshInit(22, "idrsa", "fake-host", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Nil(t, gc)
	assert.Contains(t, err.Error(), "Unable to get Public Key from Private one")
}

func TestAddRemoteLinkToGerrit_ShouldPass(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Error("Unable to create test git repo")
	}
	if _, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.com"},
	}); err != nil {
		t.Error("Unable to create remote")
	}

	err = AddRemoteLinkToGerrit(dir, "fake-host", 22, "appName", logr.DiscardLogger{})
	assert.NoError(t, err)
	b, err := os.ReadFile(fmt.Sprintf("%v/.git/config", dir))
	if err != nil {
		t.Error("Unable to read test file with git config")
	}
	assert.Contains(t, string(b), "[remote \"origin\"]\n\turl = ssh://fake-host:22/appName")
}

func TestAddRemoteLinkToGerrit_ShouldPassWithErrRemoteNotFound(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	if _, err = git.PlainInit(dir, false); err != nil {
		t.Error("Unable to create test git repo")
	}

	err = AddRemoteLinkToGerrit(dir, "fake-host", 22, "appName", logr.DiscardLogger{})
	assert.NoError(t, err)
	b, err := os.ReadFile(fmt.Sprintf("%v/.git/config", dir))
	if err != nil {
		t.Error("Unable to read test file with git config")
	}
	assert.Contains(t, string(b), "[remote \"origin\"]\n\turl = ssh://fake-host:22/appName")
}

func TestAddRemoteLinkToGerrit_ShouldFail(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	if _, err = git.PlainInit(dir, false); err != nil {
		t.Error("Unable to create test git repo")
	}

	err = AddRemoteLinkToGerrit(dir, "fake-host", 22, "appName", logr.DiscardLogger{})
	assert.NoError(t, err)
	b, err := os.ReadFile(fmt.Sprintf("%v/.git/config", dir))
	if err != nil {
		t.Error("Unable to read test file with git config")
	}
	assert.Contains(t, string(b), "[remote \"origin\"]\n\turl = ssh://fake-host:22/appName")

}

func TestAddRemoteLinkToGerrit_ShouldFailToOpenGit(t *testing.T) {
	err := AddRemoteLinkToGerrit("/tmp/1", "fake-host", 22, "appName", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to open Git directory")
}

func TestSetupProjectReplication_ShouldFailToReloadGerritPlugin(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	cmg := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "fake-namespace",
		},
		Data: map[string]string{
			"replication.config": "stub-config",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cmg)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cmg).Build()

	err := os.Setenv("ASSETS_DIR", "../../build")
	require.NoError(t, err)

	err = SetupProjectReplication(fakeCl, 22, "gerrit", idrsa, "fake-name",
		"fake-namespace", "vcs", logr.DiscardLogger{})
	//TODO: mock sshclient and implement test that passes
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "failed to dial:") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestSetupProjectReplication_ShouldFailToGetReplicationConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	err := os.Setenv("ASSETS_DIR", "/tmp")
	require.NoError(t, err)

	err = SetupProjectReplication(fakeCl, 22, "gerrit", "idrsa", "fake-name",
		"fake-namespace", "vcs", logr.DiscardLogger{})
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "Uable to generate replication config") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestSetupProjectReplication_ShouldFailToGetConfigmap(t *testing.T) {
	cmg := &coreV1.ConfigMap{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cmg)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cmg).Build()

	err := os.Setenv("ASSETS_DIR", "../../build")
	require.NoError(t, err)

	err = SetupProjectReplication(fakeCl, 22, "gerrit", "idrsa", "fake-name",
		"fake-namespace", "vcs", logr.DiscardLogger{})

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "couldn't get gerrit config map") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestSetupProjectReplication_ShouldFailToParseConfigmap(t *testing.T) {
	cmg := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "fake-namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cmg)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cmg).Build()

	err := os.Setenv("ASSETS_DIR", "../../build")
	require.NoError(t, err)

	err = SetupProjectReplication(fakeCl, 22, "gerrit", "idrsa", "fake-name",
		"fake-namespace", "vcs", logr.DiscardLogger{})

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "replication.config key is missing in gerrit ConfigMap") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReloadReplicationPlugin_ShouldFailToParseRSAKey(t *testing.T) {
	err := reloadReplicationPlugin(22, "wrong-format-pkey", "host", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to get Public Key from Private one")
}

func TestCreateProject_ShouldFailToParseRSAKey(t *testing.T) {
	err := CreateProject(22, "wrong-format-pkey", "host", "appName", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to get Public Key from Private one")
}

func TestCreateProject_ShouldFailToRunCommand(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	err := CreateProject(22, idrsa, "host", "appName", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to dial")
}

func TestSetHeadToBranch_ShouldFailToParseRSAKey(t *testing.T) {
	err := SetHeadToBranch(22, "wrong-format-pkey", "host", "appName", "defBranch", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to get Public Key from Private one")
}

func TestSetHeadToBranch_ShouldFailToRunCommand(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	err := SetHeadToBranch(22, idrsa, "host", "appName", "defBranch", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to dial")
}

func TestCheckProjectExist_ShouldFailToParseRSAKey(t *testing.T) {
	e, err := CheckProjectExist(22, "wrong-format-pkey", "host", "appName", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Nil(t, e)
	assert.Contains(t, err.Error(), "Unable to get Public Key from Private one")
}

func TestCheckProjectExist_ShouldFailToRunCommand(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	e, err := CheckProjectExist(22, idrsa, "host", "appName", logr.DiscardLogger{})
	assert.Error(t, err)
	assert.Nil(t, e)
	assert.Contains(t, err.Error(), "failed to dial")
}
