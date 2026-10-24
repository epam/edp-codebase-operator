package v2

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	corev1 "k8s.io/api/core/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))

	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		m[k] = v
	}

	return m
}

// gitConfigMap returns the GIT_CONFIG_COUNT/KEY_n/VALUE_n entries of env.
func gitConfigMap(t *testing.T, env []string) map[string]string {
	t.Helper()

	m := envMap(env)
	count, err := strconv.Atoi(m["GIT_CONFIG_COUNT"])
	require.NoError(t, err)

	cfg := make(map[string]string, count)

	for i := range count {
		cfg[m[fmt.Sprintf("GIT_CONFIG_KEY_%d", i)]] = m[fmt.Sprintf("GIT_CONFIG_VALUE_%d", i)]
	}

	return cfg
}

func TestCLIGitProvider_buildEnv_SSH(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "/etc/known_hosts")
	setCADefaults(t, nil, nil)

	key := testSSHKey(t)
	p := NewCLIGitProvider(Config{SSHKey: key, SSHUser: "gerrit", Token: "tok"})

	env, cleanup, err := p.buildEnv("")
	require.NoError(t, err)

	m := envMap(env)
	cmd := m["GIT_SSH_COMMAND"]

	assert.Contains(t, cmd, "-o StrictHostKeyChecking=yes")
	assert.Contains(t, cmd, "-o IdentitiesOnly=yes")
	assert.Contains(t, cmd, "-o BatchMode=yes")
	assert.Contains(t, cmd, `'UserKnownHostsFile="/etc/known_hosts"'`)
	assert.Contains(t, cmd, "-l 'gerrit'")
	assert.Contains(t, cmd, "HostKeyAlgorithms=+ssh-rsa")
	assert.Contains(t, cmd, "PubkeyAcceptedAlgorithms=+ssh-rsa")
	assert.NotContains(t, gitConfigMap(t, env), "http.extraHeader", "SSH key wins over token")
	assert.NotContains(t, m, "GIT_SSL_CAINFO", "SSH needs no CA bundle")
	assert.Equal(t, "0", m["GIT_TERMINAL_PROMPT"])
	assert.Equal(t, "1", m["GIT_CONFIG_NOSYSTEM"])
	assert.Equal(t, os.TempDir(), m["HOME"])

	keyFile := strings.Split(strings.Split(cmd, "-i '")[1], "'")[0]
	info, err := os.Stat(keyFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	data, err := os.ReadFile(keyFile)
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(key)+"\n", string(data))

	cleanup()

	_, err = os.Stat(keyFile)
	assert.True(t, os.IsNotExist(err))
}

func TestCLIGitProvider_buildEnv_Token(t *testing.T) {
	setCADefaults(t, nil, nil)

	tests := []struct {
		name     string
		cfg      Config
		wantAuth string
	}{
		{
			name:     "github",
			cfg:      Config{GitProvider: codebaseApi.GitProviderGithub, Username: "u", Token: "s3cret"},
			wantAuth: "u:s3cret",
		},
		{
			name:     "gitlab",
			cfg:      Config{GitProvider: codebaseApi.GitProviderGitlab, Username: "u", Token: "s3cret"},
			wantAuth: "oauth2:s3cret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, cleanup, err := NewCLIGitProvider(tt.cfg).buildEnv("https://git.example.com:8443/org/repo.git")
			require.NoError(t, err)

			defer cleanup()

			cfg := gitConfigMap(t, env)

			assert.Equal(t, "Authorization: Basic "+basicAuth(strings.Split(tt.wantAuth, ":")[0], "s3cret"),
				cfg["http.https://git.example.com:8443/.extraHeader"])
			assert.NotContains(t, cfg, "http.extraHeader", "token must be scoped to the remote host")
			assert.NotContains(t, envMap(env), "GIT_SSH_COMMAND")
		})
	}
}

func TestCLIGitProvider_buildEnv_Anonymous(t *testing.T) {
	setCADefaults(t, nil, nil)

	env, cleanup, err := NewCLIGitProvider(Config{}).buildEnv("")
	require.NoError(t, err)

	defer cleanup()

	m := envMap(env)
	assert.NotContains(t, m, "GIT_SSH_COMMAND")
	assert.NotContains(t, gitConfigMap(t, env), "http.extraHeader")
	assert.Equal(t, "0", m["GIT_TERMINAL_PROMPT"])
}

func TestBaseEnv_Config(t *testing.T) {
	m := envMap(gitEnv())

	assert.Equal(t, os.DevNull, m["GIT_CONFIG_GLOBAL"])
	assert.Equal(t, "1", m["GIT_CONFIG_NOSYSTEM"])
	assert.Equal(t, map[string]string{
		"gc.auto":          "0",
		"maintenance.auto": "false",
		"pack.threads":     "1",
	}, gitConfigMap(t, gitEnv()))
}

func TestBaseEnv_Proxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://upper:3128")
	t.Setenv("http_proxy", "http://lower:3128")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("https_proxy", "http://only-lower:3128")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	m := envMap(gitEnv())

	assert.Equal(t, "http://upper:3128", m["HTTP_PROXY"], "Go prefers the uppercase name")
	assert.Equal(t, "http://upper:3128", m["http_proxy"], "libcurl reads the lowercase name")
	assert.Equal(t, "http://only-lower:3128", m["HTTPS_PROXY"])
	assert.Equal(t, "http://only-lower:3128", m["https_proxy"])
	assert.NotContains(t, m, "NO_PROXY")
	assert.NotContains(t, m, "no_proxy")
}

func TestCLIGitProvider_redact(t *testing.T) {
	p := NewCLIGitProvider(Config{Username: "u", Token: "s3cret", SSHKey: "KEYDATA"})
	out := p.redact("fail s3cret " + basicAuth("u", "s3cret") + " KEYDATA")

	assert.NotContains(t, out, "s3cret")
	assert.NotContains(t, out, "KEYDATA")
	assert.NotContains(t, out, basicAuth("u", "s3cret"))
}

func TestCLIError_HostKey(t *testing.T) {
	err := remoteErr(cliError(assert.AnError, "Host key verification failed."), "ssh://git.example.com:29418/repo")
	assert.ErrorContains(t, err, "not present in known_hosts")
	assert.ErrorContains(t, err, "git.example.com:29418")

	err = remoteErr(cliError(assert.AnError, "REMOTE HOST IDENTIFICATION HAS CHANGED!\nHost key verification failed."),
		"git@git.example.com:repo.git")
	assert.ErrorContains(t, err, "host key mismatch")
}

func TestProviderFactories_ReturnCLIProvider(t *testing.T) {
	assert.IsType(t, &CLIGitProvider{}, NewGitProviderFactory(Config{}))
	assert.IsType(t, &CLIGitProvider{}, DefaultGitProviderFactory(&codebaseApi.GitServer{}, &corev1.Secret{}))
}

func gitRun(t *testing.T, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)

	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	return strings.TrimSpace(string(out))
}

func TestCLIGitProvider_CloneAndPush(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	setCADefaults(t, nil, nil)

	ctx := context.Background()
	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	bare := filepath.Join(root, "remote.git")

	gitRun(t, "init", "-b", "main", seed)
	require.NoError(t, os.WriteFile(filepath.Join(seed, "a.txt"), []byte("a"), 0o600))
	gitRun(t, "-C", seed, "add", ".")
	gitRun(t, "-C", seed, "commit", "-m", "init")
	gitRun(t, "-C", seed, "branch", "feature")
	gitRun(t, "-C", seed, "tag", "v1")
	gitRun(t, "clone", "--bare", seed, bare)

	url := "file://" + bare
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Clone(ctx, url, dst))

	assert.Equal(t, "main", gitRun(t, "-C", dst, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(t, "origin", gitRun(t, "-C", dst, "remote"))
	assert.Contains(t, gitRun(t, "-C", dst, "for-each-ref", "--format=%(refname)", "refs/heads"), "refs/heads/feature")
	assert.Equal(t, "v1", gitRun(t, "-C", dst, "tag", "--list"))
	assert.FileExists(t, filepath.Join(dst, "a.txt"))

	gitRun(t, "-C", dst, "branch", "new-branch")
	require.NoError(t, p.Push(ctx, dst, RefSpecPushAllBranches))
	assert.Contains(t, gitRun(t, "-C", bare, "branch", "--list"), "new-branch")

	// Re-push is a no-op success.
	require.NoError(t, p.Push(ctx, dst, RefSpecPushAllBranches, RefSpecPushAllTags))

	// Non-fast-forward is rejected.
	gitRun(t, "-C", dst, "checkout", "-q", "new-branch")
	require.NoError(t, os.WriteFile(filepath.Join(dst, "b.txt"), []byte("b"), 0o600))
	gitRun(t, "-C", dst, "add", ".")
	gitRun(t, "-C", dst, "commit", "-m", "b")
	require.NoError(t, p.Push(ctx, dst, "refs/heads/new-branch:refs/heads/new-branch"))
	gitRun(t, "-C", dst, "reset", "--hard", "-q", "HEAD~1")
	require.Error(t, p.Push(ctx, dst, "refs/heads/new-branch:refs/heads/new-branch"))
}

func TestCLIGitProvider_Clone_CloneFailureRemovesDestination(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	setCADefaults(t, nil, nil)

	root := t.TempDir()
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	err := p.Clone(context.Background(), "file://"+filepath.Join(root, "missing.git"), dst)

	require.Error(t, err)
	assert.NoDirExists(t, dst)
}

func TestCLIGitProvider_Clone_BranchCreationFailureRemovesDestination(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}

	setCADefaults(t, nil, nil)

	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	bare := filepath.Join(root, "remote.git")

	gitRun(t, "init", "-b", "main", seed)
	require.NoError(t, os.WriteFile(filepath.Join(seed, "a.txt"), []byte("a"), 0o600))
	gitRun(t, "-C", seed, "add", ".")
	gitRun(t, "-C", seed, "commit", "-m", "init")
	gitRun(t, "clone", "--bare", seed, bare)

	binDir := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = fetch ] && exit 1; done\nexec " + realGit + " \"$@\"\n"
	stub := filepath.Join(binDir, "git")
	require.NoError(t, os.WriteFile(stub, []byte(script), 0o700)) //nolint:gosec // stub must be executable
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	err = p.Clone(context.Background(), "file://"+bare, dst)

	require.ErrorContains(t, err, "failed to create local branches")
	assert.NoDirExists(t, dst)
}

func TestCLIGitProvider_buildEnv_TokenScope(t *testing.T) {
	setCADefaults(t, nil, nil)

	for _, remote := range []string{"", "git@git.example.com:org/repo.git", "ssh://git.example.com/repo", "/srv/repo"} {
		env, cleanup, err := NewCLIGitProvider(Config{Username: "u", Token: "tok"}).buildEnv(remote)
		require.NoError(t, err)

		for key := range gitConfigMap(t, env) {
			assert.NotContains(t, key, "extraHeader", "remote %q", remote)
		}

		cleanup()
	}
}

func TestCLIGitProvider_buildEnv_StallBounds(t *testing.T) {
	setCADefaults(t, nil, nil)

	env, cleanup, err := NewCLIGitProvider(Config{}).buildEnv("https://git.example.com/repo")
	require.NoError(t, err)

	defer cleanup()

	cfg := gitConfigMap(t, env)
	assert.Equal(t, "1", cfg["http.lowSpeedLimit"])
	assert.Equal(t, "600", cfg["http.lowSpeedTime"])

	assert.NotContains(t, gitConfigMap(t, gitEnv()), "http.lowSpeedLimit", "local calls have no network bound")

	sshEnv, sshCleanup, err := NewCLIGitProvider(Config{SSHKey: testSSHKey(t)}).buildEnv("")
	require.NoError(t, err)

	defer sshCleanup()

	cmd := envMap(sshEnv)["GIT_SSH_COMMAND"]
	assert.Contains(t, cmd, "ConnectTimeout=30")
	assert.Contains(t, cmd, "ServerAliveInterval=15")
	assert.Contains(t, cmd, "ServerAliveCountMax=4")
}

func TestCLIGitProvider_sshCommand_KnownHostsList(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "/etc/a"+string(os.PathListSeparator)+"/etc/b")

	cmd := NewCLIGitProvider(Config{}).sshCommand("/k")
	assert.Contains(t, cmd, `'UserKnownHostsFile="/etc/a" "/etc/b"'`)
}

func TestCLIGitProvider_buildEnv_AllowedProtocols(t *testing.T) {
	setCADefaults(t, nil, nil)

	for _, cfg := range []Config{{}, {SSHKey: testSSHKey(t)}, {Token: "tok"}} {
		env, cleanup, err := NewCLIGitProvider(cfg).buildEnv("https://git.example.com/repo")
		require.NoError(t, err)

		assert.Equal(t, allowedProtocols, envMap(env)["GIT_ALLOW_PROTOCOL"])

		cleanup()
	}

	assert.NotContains(t, envMap(gitEnv()), "GIT_ALLOW_PROTOCOL")
}

func TestCLIGitProvider_sshCommand_GlobalKnownHosts(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "/etc/known_hosts")

	cmd := NewCLIGitProvider(Config{}).sshCommand("/k")
	assert.Contains(t, cmd, "'GlobalKnownHostsFile="+os.DevNull+"'")

	t.Setenv("SSH_KNOWN_HOSTS", "")

	cmd = NewCLIGitProvider(Config{}).sshCommand("/k")
	assert.NotContains(t, cmd, "GlobalKnownHostsFile")
}

func TestCliError_Revoked(t *testing.T) {
	err := cliError(errors.New("exit status 128"),
		"@ WARNING: REVOKED HOST KEY DETECTED! @\nHost key verification failed.")

	var revoked *knownhosts.RevokedError

	require.ErrorAs(t, err, &revoked)

	var keyErr *knownhosts.KeyError

	assert.False(t, errors.As(err, &keyErr))
}

func testSSHKey(t *testing.T) string {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	block, err := ssh.MarshalPrivateKey(priv, "")
	require.NoError(t, err)

	return string(pem.EncodeToMemory(block))
}

func TestCLIGitProvider_buildEnv_InvalidSSHKey(t *testing.T) {
	_, cleanup, err := NewCLIGitProvider(Config{SSHKey: "not-a-key"}).buildEnv("")
	defer cleanup()

	require.ErrorContains(t, err, "failed to parse SSH private key")
}

func TestQuoteHostsFiles(t *testing.T) {
	assert.Equal(t, `"/etc/my certs/kh" "/x%%y"`, quoteHostsFiles([]string{"/etc/my certs/kh", "/x%y"}))
}

func TestDefaultAllowedProtocols(t *testing.T) {
	assert.Equal(t, "http:https:ssh", defaultAllowedProtocols)
}

func TestCLIGitProvider_buildEnv_KeylessSSHRemote(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "/etc/known_hosts")
	setCADefaults(t, nil, nil)

	for _, remote := range []string{
		"ssh://alice@git.example.com:2222/org/repo.git",
		"git@git.example.com:org/repo.git",
	} {
		env, cleanup, err := NewCLIGitProvider(Config{}).buildEnv(remote)
		require.NoError(t, err, remote)

		cmd := envMap(env)["GIT_SSH_COMMAND"]

		assert.Contains(t, cmd, "-o StrictHostKeyChecking=yes", remote)
		assert.Contains(t, cmd, "-o BatchMode=yes", remote)
		assert.Contains(t, cmd, `'UserKnownHostsFile="/etc/known_hosts"'`, remote)
		assert.Contains(t, cmd, "ServerAliveInterval=", remote)
		assert.NotContains(t, cmd, " -i ", remote)
		assert.NotContains(t, cmd, " -l ", remote, "the URL user applies")
		assert.Contains(t, envMap(env), "GIT_ALLOW_PROTOCOL", remote)

		cleanup()
	}
}

func TestCLIGitProvider_buildEnv_HTTPRemoteHasNoSSHCommand(t *testing.T) {
	setCADefaults(t, nil, nil)

	env, cleanup, err := NewCLIGitProvider(Config{Token: "tok"}).buildEnv("https://git.example.com/org/repo.git")
	require.NoError(t, err)

	defer cleanup()

	assert.NotContains(t, envMap(env), "GIT_SSH_COMMAND")
}
