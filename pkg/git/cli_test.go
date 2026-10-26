package v2

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ssh2 "golang.org/x/crypto/ssh"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/sshhostkey"
)

// testSSHKey returns a new ed25519 private key in the OpenSSH format.
func testSSHKey(t *testing.T) string {
	t.Helper()

	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	block, err := ssh2.MarshalPrivateKey(key, "")
	require.NoError(t, err)

	return string(pem.EncodeToMemory(block))
}

// testCertPEM returns a new self-signed CA certificate.
func testCertPEM(t *testing.T) []byte {
	t.Helper()

	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// hermeticGit keeps the developer's git configuration out of the CLI tests.
func hermeticGit(t *testing.T) {
	t.Helper()

	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// onPath puts an executable script named name first on PATH.
func onPath(t *testing.T, name, script string) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o700))
	t.Setenv("PATH", dir+string(filepath.ListSeparator)+os.Getenv("PATH"))
}

// onPathGit puts a git stand-in first on PATH; the script reaches the real git as $REAL_GIT.
func onPathGit(t *testing.T, script string) {
	t.Helper()

	realGit, err := exec.LookPath("git")
	require.NoError(t, err)

	t.Setenv("REAL_GIT", realGit)
	onPath(t, "git", script)
}

// initOrigin creates a bare repository with the branches master and feature and the tag v1.
func initOrigin(t *testing.T, dir string) {
	t.Helper()

	repo, err := gogit.PlainInit(dir, true)
	require.NoError(t, err)

	master := seedCommit(t, dir, "base", "+refs/heads/master:refs/heads/master")
	seedCommit(t, dir, "feature", "+refs/heads/master:refs/heads/feature")

	_, err = repo.CreateTag("v1", master, nil)
	require.NoError(t, err)
}

// localRefs lists the branches and tags of a repository.
func localRefs(t *testing.T, dir string) []string {
	t.Helper()

	repo, err := gogit.PlainOpen(dir)
	require.NoError(t, err)

	iter, err := repo.References()
	require.NoError(t, err)

	var refs []string

	require.NoError(t, iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() || ref.Name().IsTag() {
			refs = append(refs, ref.Name().String())
		}

		return nil
	}))

	slices.Sort(refs)

	return refs
}

// runGit runs git in dir and returns its output.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	require.NoError(t, err, string(out))

	return string(out)
}

// gitBackend serves the bare repositories under root through git http-backend.
func gitBackend(t *testing.T, root, user string) http.Handler {
	t.Helper()

	execPath, err := exec.Command("git", "--exec-path").Output()
	require.NoError(t, err)

	return &cgi.Handler{
		Path: filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend"),
		// REMOTE_USER makes http-backend accept pushes; PATH stays the one without
		// the stand-ins that tests put on it later.
		Env: []string{
			"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=" + user,
			"GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "PATH=" + os.Getenv("PATH"),
		},
	}
}

// gitHTTPServer serves the bare repositories under root over HTTPS through
// git http-backend, behind Basic authentication.
func gitHTTPServer(t *testing.T, root, user, password string) *httptest.Server {
	t.Helper()

	backend := gitBackend(t, root, user)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != user || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestNewGitProviderFactory(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		unset   bool
		wantCLI bool
	}{
		{name: "unset selects the git binary", unset: true, wantCLI: true},
		{name: "true selects the git binary", value: "true", wantCLI: true},
		{name: "false selects go-git", value: "false", wantCLI: false},
		{name: "an invalid value keeps the default", value: "maybe", wantCLI: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(CLIEnabledEnv, tt.value)

			if tt.unset {
				require.NoError(t, os.Unsetenv(CLIEnabledEnv))
			}

			_, isCLI := NewGitProviderFactory(Config{}).(*CLIProvider)
			assert.Equal(t, tt.wantCLI, isCLI)
			assert.Equal(t, tt.wantCLI, CLIEnabled())
		})
	}
}

func TestCLIProvider_Clone(t *testing.T) {
	hermeticGit(t)

	origin := t.TempDir()
	initOrigin(t, origin)

	clone := func(p Git) ([]string, string) {
		dir := filepath.Join(t.TempDir(), "clone")
		require.NoError(t, p.Clone(t.Context(), origin, dir))
		assert.Empty(t, runGit(t, dir, "status", "--porcelain"), "the working tree must match HEAD")

		return localRefs(t, dir), workdirHead(t, dir).Name().Short()
	}

	wantRefs, wantHead := clone(NewGitProvider(Config{}))
	gotRefs, gotHead := clone(NewCLIProvider(Config{}))

	assert.Equal(t, []string{"refs/heads/feature", "refs/heads/master", "refs/tags/v1"}, wantRefs)
	assert.Equal(t, wantRefs, gotRefs, "the CLI clone must hold the same local refs as the go-git clone")
	assert.Equal(t, wantHead, gotHead)
}

// A push to the default branch while the clone runs must not leave the working
// tree behind the branch, or the next commit would revert that push.
func TestCLIProvider_Clone_BranchMovedDuringClone(t *testing.T) {
	hermeticGit(t)

	origin := t.TempDir()
	initOrigin(t, origin)

	moved := seedCommit(t, origin, "moved", "+refs/heads/master:refs/keep/moved")

	// After the real clone, origin's master moves to a commit the clone has not seen.
	t.Setenv("ORIGIN", origin)
	t.Setenv("MOVED", moved.String())
	onPathGit(t, `#!/bin/sh
"$REAL_GIT" "$@" || exit
[ "$1" = clone ] && "$REAL_GIT" --git-dir="$ORIGIN" update-ref refs/heads/master "$MOVED"
exit 0
`)

	dir := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, NewCLIProvider(Config{}).Clone(t.Context(), origin, dir))

	assert.Equal(t, moved, workdirHead(t, dir).Hash())
	assert.Empty(t, runGit(t, dir, "status", "--porcelain"), "the working tree must match the moved branch")

	content, err := os.ReadFile(filepath.Join(dir, "f.txt"))
	require.NoError(t, err)
	assert.Equal(t, "moved", string(content))
}

// go-git commits on the CLI checkout later and applies no .gitattributes
// conversion, so the checkout must not apply one either.
func TestCLIProvider_Clone_NoAttributeConversion(t *testing.T) {
	hermeticGit(t)

	origin := t.TempDir()
	_, err := gogit.PlainInit(origin, true)
	require.NoError(t, err)

	seed := t.TempDir()
	runGit(t, seed, "init", "--quiet", "-b", "master")
	require.NoError(t, os.WriteFile(filepath.Join(seed, ".gitattributes"), []byte("*.bat text eol=crlf\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(seed, "gradlew.bat"), []byte("echo one\necho two\n"), 0o600))
	runGit(t, seed, "add", ".")
	runGit(t, seed, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-m", "init")
	runGit(t, seed, "push", "--quiet", origin, "master")

	dir := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, NewCLIProvider(Config{}).Clone(t.Context(), origin, dir))

	content, err := os.ReadFile(filepath.Join(dir, "gradlew.bat"))
	require.NoError(t, err)
	assert.Equal(t, "echo one\necho two\n", string(content), "the file must hold the committed bytes, as with go-git")
}

func TestCLIProvider_Push(t *testing.T) {
	hermeticGit(t)

	tests := []struct {
		name     string
		refspecs []string
		want     []string
	}{
		{
			name: "no refspec pushes every branch, as go-git does",
			want: []string{"refs/heads/feature", "refs/heads/master"},
		},
		{
			name:     "branches and tags",
			refspecs: []string{RefSpecPushAllBranches, RefSpecPushAllTags},
			want:     []string{"refs/heads/feature", "refs/heads/master", "refs/tags/v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewCLIProvider(Config{})

			origin := t.TempDir()
			initOrigin(t, origin)

			workDir := filepath.Join(t.TempDir(), "clone")
			require.NoError(t, p.Clone(t.Context(), origin, workDir))

			target := t.TempDir()
			_, err := gogit.PlainInit(target, true)
			require.NoError(t, err)
			require.NoError(t, p.AddRemoteLink(t.Context(), workDir, target))

			require.NoError(t, p.Push(t.Context(), workDir, tt.refspecs...))
			assert.Equal(t, tt.want, localRefs(t, target))

			require.NoError(t, p.Push(t.Context(), workDir, tt.refspecs...), "an up-to-date push is not an error")
		})
	}
}

func TestCLIProvider_Errors(t *testing.T) {
	hermeticGit(t)

	p := NewCLIProvider(Config{})

	t.Run("missing repository", func(t *testing.T) {
		err := p.Clone(t.Context(), filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "clone"))
		require.ErrorContains(t, err, "failed to clone repository")
	})

	t.Run("a URL is never read as an option", func(t *testing.T) {
		// Read as an option, this URL would make git clone the destination with "touch" as upload-pack.
		origin := t.TempDir()
		initOrigin(t, origin)

		marker := filepath.Join(t.TempDir(), "marker")
		require.Error(t, p.Clone(t.Context(), "--upload-pack=touch "+marker, origin))
		assert.NoFileExists(t, marker)
	})

	t.Run("empty repository fails as with go-git and leaves no clone", func(t *testing.T) {
		origin := t.TempDir()
		_, err := gogit.PlainInit(origin, true)
		require.NoError(t, err)

		for _, gp := range []Git{NewGitProvider(Config{}), p} {
			dir := filepath.Join(t.TempDir(), "clone")
			require.ErrorIs(t, gp.Clone(t.Context(), origin, dir), transport.ErrEmptyRemoteRepository)
			assert.NoDirExists(t, dir)
		}
	})

	t.Run("a failed fetch leaves no partial clone", func(t *testing.T) {
		onPathGit(t, "#!/bin/sh\n[ \"$1\" = fetch ] && exit 1\nexec \"$REAL_GIT\" \"$@\"\n")

		origin := t.TempDir()
		initOrigin(t, origin)

		dir := filepath.Join(t.TempDir(), "clone")
		require.ErrorContains(t, p.Clone(t.Context(), origin, dir), "failed to fetch all branches and tags")
		assert.NoDirExists(t, dir)
	})

	t.Run("a killed clone leaves no partial clone", func(t *testing.T) {
		// git killed mid-clone never runs its own cleanup and leaves the .git it has written.
		onPathGit(t, "#!/bin/sh\nif [ \"$1\" = clone ]; then for last; do :; done; mkdir -p \"$last/.git\"; kill -9 $$; fi\n"+
			"exec \"$REAL_GIT\" \"$@\"\n")

		origin := t.TempDir()
		initOrigin(t, origin)

		dir := filepath.Join(t.TempDir(), "clone")
		require.ErrorContains(t, p.Clone(t.Context(), origin, dir), "failed to clone repository")
		assert.NoDirExists(t, dir)
	})

	t.Run("a HEAD that names a missing branch is not an empty repository", func(t *testing.T) {
		origin := t.TempDir()
		initOrigin(t, origin)
		runGit(t, origin, "update-ref", "-d", "refs/heads/master")

		dir := filepath.Join(t.TempDir(), "clone")
		err := p.Clone(t.Context(), origin, dir)
		require.ErrorIs(t, err, plumbing.ErrReferenceNotFound)
		assert.NotErrorIs(t, err, transport.ErrEmptyRemoteRepository)
		assert.NoDirExists(t, dir)
	})

	t.Run("rejected push", func(t *testing.T) {
		origin := t.TempDir()
		initOrigin(t, origin)

		workDir := filepath.Join(t.TempDir(), "clone")
		require.NoError(t, p.Clone(t.Context(), origin, workDir))

		seedCommit(t, origin, "unrelated", "+refs/heads/master:refs/heads/master")
		require.ErrorContains(t, p.Push(t.Context(), workDir, RefSpecPushAllBranches), "rejected")
	})

	t.Run("push outside a repository", func(t *testing.T) {
		require.ErrorContains(t, p.Push(t.Context(), t.TempDir()), "failed to open repository")
	})
}

func TestCLIProvider_HTTPS(t *testing.T) {
	hermeticGit(t)

	const token = "glpat-secret"

	root := t.TempDir()
	initOrigin(t, filepath.Join(root, "repo.git"))

	srv := gitHTTPServer(t, root, "oauth2", token)
	repoURL := srv.URL + "/repo.git"

	trusted := t.TempDir()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(filepath.Join(trusted, "ca.crt"), certPEM, 0o600))

	tests := []struct {
		name      string
		token     string
		certDir   string
		env       map[string]string
		wantErr   string
		wantErrIs error
	}{
		{name: "token over a CA from SSL_CERT_DIR", token: token, certDir: trusted},
		{
			name: "rejected token", token: "glpat-wrong", certDir: trusted,
			wantErr: "failed to clone repository", wantErrIs: transport.ErrAuthenticationRequired,
		},
		{name: "CA missing from SSL_CERT_DIR", token: token, wantErr: "certificate"},
		{
			name: "GIT_SSL_NO_VERIFY does not turn verification off", token: token,
			env: map[string]string{"GIT_SSL_NO_VERIFY": "1"}, wantErr: "certificate",
		},
		{
			name: "GIT_CONFIG_PARAMETERS does not turn verification off", token: token,
			env: map[string]string{"GIT_CONFIG_PARAMETERS": "'http.sslverify'='false'"}, wantErr: "certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SSL_CERT_DIR", tt.certDir)

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			argv := filepath.Join(t.TempDir(), "argv")
			t.Setenv("ARGV", argv)
			onPathGit(t, "#!/bin/sh\necho \"$@\" >> \"$ARGV\"\nexec \"$REAL_GIT\" \"$@\"\n")

			p := NewCLIProvider(Config{GitProvider: codebaseApi.GitProviderGitlab, Token: tt.token})
			workDir := filepath.Join(t.TempDir(), "clone")
			secrets := []string{tt.token, base64.StdEncoding.EncodeToString([]byte("oauth2:" + tt.token))}

			err := p.Clone(t.Context(), repoURL, workDir)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.NoError(t, p.CreateChildBranch(t.Context(), workDir, "master", "pushed"))
				require.NoError(t, p.Push(t.Context(), workDir, RefSpecPushAllBranches))
				assert.Contains(t, localRefs(t, filepath.Join(root, "repo.git")), "refs/heads/pushed")
			} else {
				require.ErrorContains(t, err, tt.wantErr)

				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}

				for _, secret := range secrets {
					assert.NotContains(t, err.Error(), secret)
				}
			}

			args, err := os.ReadFile(argv)
			require.NoError(t, err)

			for _, secret := range secrets {
				assert.NotContains(t, string(args), secret, "credentials must never be in git's arguments")
			}
		})
	}
}

// The SSH path, with an ssh stand-in that records its arguments and the key
// file's mode, then runs the remote git command locally.
func TestCLIProvider_SSH(t *testing.T) {
	hermeticGit(t)
	t.Setenv("SSH_KNOWN_HOSTS", "/etc/codebase-operator/ssh/ssh_known_hosts")

	argv := filepath.Join(t.TempDir(), "argv")
	t.Setenv("ARGV", argv)
	onPath(t, "ssh", `#!/bin/sh
for arg; do
  echo "$arg" >> "$ARGV"
  [ "$prev" = -i ] && echo "key mode $(stat -c %a "$arg" 2>/dev/null || stat -f %Lp "$arg")" >> "$ARGV"
  prev=$arg
done
exec sh -c "$arg"
`)

	origin := t.TempDir()
	initOrigin(t, origin)

	// Like a Gerrit URL: no user in it.
	repoURL := "ssh://git.example.com:29418" + filepath.ToSlash(origin)
	p := NewCLIProvider(Config{SSHKey: testSSHKey(t), SSHUser: "edp-ci"})

	workDir := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, p.Clone(t.Context(), repoURL, workDir))
	require.NoError(t, p.CreateChildBranch(t.Context(), workDir, "master", "pushed"))
	require.NoError(t, p.Push(t.Context(), workDir, RefSpecPushAllBranches))
	assert.Contains(t, localRefs(t, origin), "refs/heads/pushed")

	recorded, err := os.ReadFile(argv)
	require.NoError(t, err)

	args := strings.Split(string(recorded), "\n")
	assert.Subset(t, args, []string{
		"BatchMode=yes", "IdentitiesOnly=yes", "key mode 600",
		"ConnectTimeout=30", "ServerAliveInterval=30", "ServerAliveCountMax=4",
	})
	assert.Subset(t, args, sshhostkey.OpenSSHOptions())
	assert.Equal(t, "edp-ci", args[slices.Index(args, "-l")+1], "ssh must log in as the GitServer user")
}

// A host key failure reported by ssh reaches the caller with the known_hosts guidance.
func TestCLIProvider_SSH_HostKeyFailure(t *testing.T) {
	hermeticGit(t)
	t.Setenv("SSH_KNOWN_HOSTS", "/etc/codebase-operator/ssh/ssh_known_hosts")
	onPath(t, "ssh", "#!/bin/sh\necho 'Host key verification failed.' >&2\nexit 255\n")

	p := NewCLIProvider(Config{SSHKey: testSSHKey(t), SSHUser: "edp-ci"})
	err := p.Clone(t.Context(), "ssh://gerrit.example.com:29418/proj", filepath.Join(t.TempDir(), "clone"))

	require.ErrorContains(t, err, "not present in known_hosts (/etc/codebase-operator/ssh/ssh_known_hosts)")
	require.ErrorContains(t, err, "ssh-keyscan -p 29418 gerrit.example.com")
}

// The temporary SSH key and CA bundle of a git run are gone after it, whether it succeeded or failed.
func TestCLIProvider_RemovesCredentials(t *testing.T) {
	hermeticGit(t)

	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	certs := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(certs, "ca.crt"), testCertPEM(t), 0o600))
	t.Setenv("SSL_CERT_DIR", certs)

	// What TMPDIR holds while git runs, which proves the credentials were written there.
	seen := filepath.Join(t.TempDir(), "seen")
	t.Setenv("SEEN", seen)
	onPathGit(t, "#!/bin/sh\nls -R \"$TMPDIR\" >> \"$SEEN\"\nexec \"$REAL_GIT\" \"$@\"\n")

	origin := t.TempDir()
	initOrigin(t, origin)

	p := NewCLIProvider(Config{SSHKey: testSSHKey(t), Token: "token"})
	require.NoError(t, p.Clone(t.Context(), origin, filepath.Join(t.TempDir(), "clone")))
	require.Error(t, p.Clone(t.Context(), filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "clone")))

	during, err := os.ReadFile(seen)
	require.NoError(t, err)
	assert.Contains(t, string(during), "ca-certificates.crt")
	assert.Contains(t, strings.Fields(string(during)), "id")

	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// GIT_ALLOW_PROTOCOL overrides protocol.<name>.allow, so ext:: stays refused even
// when the git configuration allows it.
func TestCLIProvider_RefusesExtTransport(t *testing.T) {
	hermeticGit(t)

	gitConfig := filepath.Join(t.TempDir(), "gitconfig")
	require.NoError(t, os.WriteFile(gitConfig, []byte("[protocol \"ext\"]\n\tallow = always\n"), 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)

	marker := filepath.Join(t.TempDir(), "marker")
	extURL := "ext::sh -c touch% " + marker + " %S x"
	err := NewCLIProvider(Config{}).Clone(t.Context(), extURL, filepath.Join(t.TempDir(), "clone"))

	require.ErrorContains(t, err, "not allowed")
	assert.NoFileExists(t, marker)
}

func TestCLIProvider_env(t *testing.T) {
	t.Setenv("SSL_CERT_DIR", "")
	t.Setenv("SSL_CERT_FILE", "")

	key := testSSHKey(t)
	tmp := t.TempDir()
	p := NewCLIProvider(Config{
		// A spelling go-git reads and ssh does not.
		SSHKey:      "\n" + strings.ReplaceAll(key, "\n", "\r\n") + "  \n",
		SSHUser:     "edp-ci",
		GitProvider: codebaseApi.GitProviderGitlab,
		Token:       "glpat-secret",
	})

	env, err := p.env(tmp, "https://git.example.com:8443/group/repo.git")
	require.NoError(t, err)

	vars := map[string]string{}

	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		vars[k] = v
	}

	count, err := strconv.Atoi(vars["GIT_CONFIG_COUNT"])
	require.NoError(t, err)

	gitConfig := make([]string, 0, count)

	for i := range count {
		key, value := vars[fmt.Sprintf("GIT_CONFIG_KEY_%d", i)], vars[fmt.Sprintf("GIT_CONFIG_VALUE_%d", i)]
		gitConfig = append(gitConfig, key+"="+value)
	}

	helper := filepath.Join(tmp, "credential-helper")
	assert.Equal(t, []string{
		"maintenance.auto=false",
		"pack.threads=2",
		"pack.windowMemory=100m",
		"core.bigFileThreshold=64m",
		"http.sslVerify=true",
		"credential.helper=",
		"credential.helper=" + helper,
	}, gitConfig)
	assert.Equal(t, "https://git.example.com:8443", vars["KRCI_GIT_REMOTE"])
	assert.Equal(t, "oauth2", vars["KRCI_GIT_USERNAME"])
	assert.Equal(t, "glpat-secret", vars["KRCI_GIT_PASSWORD"])
	assert.NotContains(t, vars, "GIT_SSL_CAINFO", "without SSL_CERT_FILE and SSL_CERT_DIR git keeps its own trust store")

	written, err := os.ReadFile(filepath.Join(tmp, "id"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(written), "-----BEGIN OPENSSH PRIVATE KEY-----\n"))
	assert.NotContains(t, string(written), "\r")

	got, err := ssh2.ParsePrivateKey(written)
	require.NoError(t, err)
	want, err := ssh2.ParsePrivateKey([]byte(key))
	require.NoError(t, err)
	assert.Equal(t, want.PublicKey().Marshal(), got.PublicKey().Marshal(), "the same key, re-encoded")

	_, err = NewCLIProvider(Config{SSHKey: "not a key"}).env(t.TempDir(), "")
	require.ErrorContains(t, err, "failed to parse SSH private key")
}

func TestCredentialHelper(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "credential-helper")
	require.NoError(t, os.WriteFile(helper, []byte(credentialHelper), 0o700))

	t.Setenv("KRCI_GIT_REMOTE", "https://git.example.com:8443")
	t.Setenv("KRCI_GIT_USERNAME", "oauth2")
	t.Setenv("KRCI_GIT_PASSWORD", "glpat-secret")

	answer := func(operation, request string) string {
		cmd := exec.Command(helper, operation)
		cmd.Stdin = strings.NewReader(request)

		out, err := cmd.Output()
		require.NoError(t, err)

		return string(out)
	}

	assert.Equal(t, "username=oauth2\npassword=glpat-secret\n",
		answer("get", "protocol=https\nhost=git.example.com:8443\npath=group/repo.git\n"))
	assert.Empty(t, answer("get", "protocol=https\nhost=other.example.com\n"), "another host")
	assert.Empty(t, answer("get", "protocol=http\nhost=git.example.com:8443\n"), "plain http")
	assert.Empty(t, answer("store", "protocol=https\nhost=git.example.com:8443\n"), "only get is answered")
}

// Credentials are scoped to the remote: after a redirect to another host git sends none, as go-git.
func TestCLIProvider_HTTPS_RedirectGetsNoCredentials(t *testing.T) {
	hermeticGit(t)

	root := t.TempDir()
	initOrigin(t, filepath.Join(root, "repo.git"))
	backend := gitBackend(t, root, "")

	var targetSawAuth atomic.Bool

	// The target asks for credentials, so git would send whatever it holds for the target's host.
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		targetSawAuth.Store(true)
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(target.Close)

	// The remote moves git to the target on the first request, before any credentials are asked for.
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.RequestURI(), http.StatusMovedPermanently)
	}))
	t.Cleanup(origin.Close)

	trusted := t.TempDir()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: origin.Certificate().Raw})
	require.NoError(t, os.WriteFile(filepath.Join(trusted, "ca.crt"), certPEM, 0o600))
	t.Setenv("SSL_CERT_DIR", trusted)

	p := NewCLIProvider(Config{GitProvider: codebaseApi.GitProviderGitlab, Token: "glpat-secret"})
	require.ErrorIs(t, p.Clone(t.Context(), origin.URL+"/repo.git", filepath.Join(t.TempDir(), "clone")),
		transport.ErrAuthenticationRequired)
	assert.False(t, targetSawAuth.Load(), "the redirect target must not get the remote's credentials")
}

func TestCLIProvider_Push_Atomic(t *testing.T) {
	hermeticGit(t)

	p := NewCLIProvider(Config{})

	setup := func(t *testing.T) (string, string) {
		t.Helper()

		origin := t.TempDir()
		initOrigin(t, origin)

		workDir := filepath.Join(t.TempDir(), "clone")
		require.NoError(t, p.Clone(t.Context(), origin, workDir))

		target := t.TempDir()
		_, err := gogit.PlainInit(target, true)
		require.NoError(t, err)
		require.NoError(t, p.AddRemoteLink(t.Context(), workDir, target))

		return workDir, target
	}

	t.Run("a rejected ref stops every ref, as with go-git", func(t *testing.T) {
		workDir, target := setup(t)
		seedCommit(t, target, "unrelated", "+refs/heads/master:refs/heads/master")

		require.Error(t, p.Push(t.Context(), workDir, RefSpecPushAllBranches))
		assert.Equal(t, []string{"refs/heads/master"}, localRefs(t, target), "feature must not land alone")
	})

	t.Run("a server without atomic push gets a plain one", func(t *testing.T) {
		workDir, target := setup(t)
		runGit(t, target, "config", "receive.advertiseAtomic", "false")

		require.NoError(t, p.Push(t.Context(), workDir, RefSpecPushAllBranches))
		assert.Equal(t, []string{"refs/heads/feature", "refs/heads/master"}, localRefs(t, target))
	})
}

// GIT_DIR or GIT_WORK_TREE in the operator's environment, which a git hook that runs these tests sets,
// must not point git at another repository.
func TestCLIProvider_IgnoresInheritedGitDir(t *testing.T) {
	hermeticGit(t)

	other := t.TempDir()
	runGit(t, other, "init", "--quiet", "-b", "master")
	require.NoError(t, os.WriteFile(filepath.Join(other, "other.txt"), []byte("other"), 0o600))
	runGit(t, other, "add", ".")
	runGit(t, other, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-m", "other")

	origin := t.TempDir()
	initOrigin(t, origin)

	head := workdirHead(t, other).Hash()

	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)

	dir := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, NewCLIProvider(Config{}).Clone(t.Context(), origin, dir))

	assert.Equal(t, []string{"refs/heads/feature", "refs/heads/master", "refs/tags/v1"}, localRefs(t, dir))
	assert.FileExists(t, filepath.Join(dir, "f.txt"))
	assert.Equal(t, head, workdirHead(t, other).Hash(), "the other repository must stay untouched")
}

func TestGitError(t *testing.T) {
	cause := errors.New("exit status 128")

	assert.Equal(t, "git push: exit status 128", gitError("push", nil, cause).Error())

	long := gitError("push", []byte(strings.Repeat("ж", maxErrorOutput)+"fatal: the end"), cause)
	assert.True(t, utf8.ValidString(long.Error()))
	assert.Contains(t, long.Error(), "git push: ...")
	assert.Contains(t, long.Error(), "fatal: the end: exit status 128")
	assert.Less(t, len(long.Error()), maxErrorOutput+64)
	assert.ErrorIs(t, long, cause)

	for _, out := range []string{
		"fatal: could not read Username for 'https://git.example.com': terminal prompts disabled",
		"fatal: Authentication failed for 'https://git.example.com/repo.git/'",
	} {
		err := gitError("clone", []byte(out), cause)
		assert.ErrorIs(t, err, transport.ErrAuthenticationRequired)
		assert.ErrorIs(t, err, cause)
	}
}

func TestShellJoin(t *testing.T) {
	args := []string{"ssh", "-l", "it's", "-o", "UserKnownHostsFile=/a b /c", "$(touch x)"}

	cmd := exec.Command("sh", "-c", `printf '%s\n' `+shellJoin(args))
	cmd.Dir = t.TempDir()

	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(args, "\n")+"\n", string(out), "the shell must get back the exact arguments")
	assert.NoFileExists(t, filepath.Join(cmd.Dir, "x"))
}

func TestCaBundle(t *testing.T) {
	t.Setenv("SSL_CERT_DIR", "")
	t.Setenv("SSL_CERT_FILE", "")

	bundle, err := caBundle(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, bundle, "without SSL_CERT_FILE and SSL_CERT_DIR git keeps its own trust store")

	cert := testCertPEM(t)
	certs := t.TempDir()
	files := map[string]string{
		"ca.crt":     string(cert),
		"tls.pem":    "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----\n" + string(cert),
		"broken.crt": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
		"README":     "not a certificate",
	}

	for name, data := range files {
		require.NoError(t, os.WriteFile(filepath.Join(certs, name), []byte(data), 0o600))
	}

	require.NoError(t, os.Mkdir(filepath.Join(certs, "..data"), 0o700))

	tests := []struct {
		name     string
		certFile string
		certDir  string
	}{
		{name: "SSL_CERT_DIR", certDir: certs + string(filepath.ListSeparator) + filepath.Join(t.TempDir(), "missing")},
		{name: "SSL_CERT_FILE", certFile: filepath.Join(certs, "ca.crt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SSL_CERT_FILE", tt.certFile)
			t.Setenv("SSL_CERT_DIR", tt.certDir)

			bundle, err := caBundle(t.TempDir())
			require.NoError(t, err)

			data, err := os.ReadFile(bundle)
			require.NoError(t, err)
			assert.Equal(t, 1, strings.Count(string(data), string(cert)), "the certificate, once")
			assert.NotContains(t, string(data), "PRIVATE KEY")

			// OpenSSL rejects the whole file for one block that does not parse.
			for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
				_, err := x509.ParseCertificate(block.Bytes)
				require.NoError(t, err)
			}
		})
	}
}
