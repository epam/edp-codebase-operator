package gerrit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func setupSuite(tb testing.TB) (func(tb testing.TB), string) {
	log.Println("setup suite")

	pk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		tb.Fatal("failed to generate test private key")
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

func TestSShInit_ShouldPass(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	g, err := SshInit(22, idrsa, "fake-host", "admin", logr.Discard())
	assert.NoError(t, err)
	assert.Equal(t, g.Config.User, "admin")
	assert.Equal(t, g.Host, "fake-host")
	assert.Equal(t, g.Port, int32(22))
}

func TestSShInit_ShouldFailForIncorrectRSAPKey(t *testing.T) {
	gc, err := SshInit(22, "idrsa", "fake-host", "admin", logr.Discard())
	assert.Error(t, err)
	assert.Nil(t, gc)
	assert.Contains(t, err.Error(), "failed to get Public Key from Private one")
}

func TestCreateProject_ShouldFailToParseRSAKey(t *testing.T) {
	g := &SSHGerritClient{}
	err := g.CreateProject(22, "wrong-format-pkey", "host", "admin", "appName", logr.Discard())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Public Key from Private one")
}

func TestCreateProject_ShouldFailToRunCommand(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	g := &SSHGerritClient{}
	err := g.CreateProject(22, idrsa, "host", "admin", "appName", logr.Discard())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to dial")
}

func TestSetHeadToBranch_ShouldFailToParseRSAKey(t *testing.T) {
	g := &SSHGerritClient{}
	err := g.SetHeadToBranch(22, "wrong-format-pkey", "host", "admin", "appName", "defBranch", logr.Discard())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Public Key from Private one")
}

func TestSetHeadToBranch_ShouldFailToRunCommand(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	g := &SSHGerritClient{}
	err := g.SetHeadToBranch(22, idrsa, "host", "admin", "appName", "defBranch", logr.Discard())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to dial")
}

func TestCheckProjectExist_ShouldFailToParseRSAKey(t *testing.T) {
	g := &SSHGerritClient{}
	e, err := g.CheckProjectExist(22, "wrong-format-pkey", "host", "admin", "appName", logr.Discard())
	assert.Error(t, err)
	assert.False(t, e)
	assert.Contains(t, err.Error(), "failed to get Public Key from Private one")
}

func TestCheckProjectExist_ShouldFailToRunCommand(t *testing.T) {
	teardownSuite, idrsa := setupSuite(t)
	defer teardownSuite(t)

	g := &SSHGerritClient{}
	e, err := g.CheckProjectExist(22, idrsa, "host", "admin", "appName", logr.Discard())
	assert.Error(t, err)
	assert.False(t, e)
	assert.Contains(t, err.Error(), "failed to dial")
}
