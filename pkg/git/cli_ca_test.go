package v2

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func certPEM(t *testing.T, cn string) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func setCADefaults(t *testing.T, files, dirs []string) {
	t.Helper()

	oldFiles, oldDirs := caFiles, caDirs
	caFiles, caDirs = files, dirs

	t.Cleanup(func() { caFiles, caDirs = oldFiles, oldDirs })
	t.Setenv("SSL_CERT_FILE", "")
	t.Setenv("SSL_CERT_DIR", "")
}

func bundleCerts(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var names []string

	for {
		var block *pem.Block

		block, data = pem.Decode(data)
		if block == nil {
			return names
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)

		names = append(names, cert.Subject.CommonName)
	}
}

func TestWriteCABundle_DefaultLocations(t *testing.T) {
	root := t.TempDir()
	certsDir := filepath.Join(root, "certs")
	require.NoError(t, os.Mkdir(certsDir, 0o700))

	system := certPEM(t, "system")
	systemFile := filepath.Join(root, "ca-certificates.crt")
	require.NoError(t, os.WriteFile(systemFile, system, 0o600))

	require.NoError(t, os.WriteFile(filepath.Join(certsDir, "gitlab-ca.crt"), certPEM(t, "mounted"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(certsDir, "copy-of-system.pem"), system, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(certsDir, "notes.txt"), []byte("ignore"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(certsDir, "broken.pem"),
		[]byte("-----BEGIN CERTIFICATE-----\nQUFB\n-----END CERTIFICATE-----\n"), 0o600))

	setCADefaults(t,
		[]string{filepath.Join(root, "missing.crt"), systemFile},
		[]string{certsDir, filepath.Join(root, "absent")},
	)

	path, err := writeCABundle()
	require.NoError(t, err)

	t.Cleanup(func() { _ = os.Remove(path) })

	assert.ElementsMatch(t, []string{"system", "mounted"}, bundleCerts(t, path))
}

func TestWriteCABundle_EnvOverridesDefaults(t *testing.T) {
	root := t.TempDir()
	defaultFile := filepath.Join(root, "default.crt")
	require.NoError(t, os.WriteFile(defaultFile, certPEM(t, "default"), 0o600))

	envFile := filepath.Join(root, "env.crt")
	require.NoError(t, os.WriteFile(envFile, certPEM(t, "env-file"), 0o600))

	envDir := filepath.Join(root, "envdir")
	require.NoError(t, os.Mkdir(envDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(envDir, "custom.crt"), certPEM(t, "env-dir"), 0o600))

	setCADefaults(t, []string{defaultFile}, []string{root})
	t.Setenv("SSL_CERT_FILE", envFile)
	t.Setenv("SSL_CERT_DIR", envDir)

	path, err := writeCABundle()
	require.NoError(t, err)

	t.Cleanup(func() { _ = os.Remove(path) })

	assert.ElementsMatch(t, []string{"env-file", "env-dir"}, bundleCerts(t, path))
}

func TestCLIGitProvider_buildEnv_NoCA(t *testing.T) {
	setCADefaults(t, []string{filepath.Join(t.TempDir(), "missing")}, nil)

	env, cleanup, err := NewCLIGitProvider(Config{}).buildEnv("https://git.example.com/repo")
	require.NoError(t, err)

	defer cleanup()

	assert.NotContains(t, envMap(env), "GIT_SSL_CAINFO")
}

func TestCLIGitProvider_buildEnv_CABundleRemoved(t *testing.T) {
	f := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(f, certPEM(t, "ca"), 0o600))
	setCADefaults(t, []string{f}, nil)

	env, cleanup, err := NewCLIGitProvider(Config{}).buildEnv("https://git.example.com/repo")
	require.NoError(t, err)

	bundle := envMap(env)["GIT_SSL_CAINFO"]
	require.FileExists(t, bundle)

	cleanup()

	assert.NoFileExists(t, bundle)
}

// TestCLIGitProvider_Clone_HTTPSCustomCA serves a repository over HTTPS with a self-signed
// certificate. The CA file sits in a trusted directory without an OpenSSL hashed name.
func TestCLIGitProvider_Clone_HTTPSCustomCA(t *testing.T) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not available")
	}

	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	gitRun(t, "init", "-b", "main", seed)
	require.NoError(t, os.WriteFile(filepath.Join(seed, "a.txt"), []byte("a"), 0o600))
	gitRun(t, "-C", seed, "add", ".")
	gitRun(t, "-C", seed, "commit", "-m", "init")
	gitRun(t, "clone", "--bare", seed, filepath.Join(root, "srv", "repo.git"))

	srv := httptest.NewTLSServer(&cgi.Handler{
		Path: gitBin,
		Args: []string{"http-backend"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + filepath.Join(root, "srv"),
			"GIT_HTTP_EXPORT_ALL=1",
		},
	})
	defer srv.Close()

	certsDir := filepath.Join(root, "certs")
	require.NoError(t, os.Mkdir(certsDir, 0o700))

	setCADefaults(t, []string{filepath.Join(root, "missing")}, []string{certsDir})

	repoURL := srv.URL + "/repo.git"
	p := NewCLIGitProvider(Config{})

	require.Error(t, p.Clone(context.Background(), repoURL, filepath.Join(root, "untrusted")),
		"clone must fail before the CA is trusted")

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(filepath.Join(certsDir, "server-ca.crt"), caPEM, 0o600))

	dst := filepath.Join(root, "trusted")
	require.NoError(t, p.Clone(context.Background(), repoURL, dst))
	assert.FileExists(t, filepath.Join(dst, "a.txt"))
}
