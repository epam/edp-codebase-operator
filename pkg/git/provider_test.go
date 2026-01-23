// nolint:dupl // Duplicate test setup is acceptable in tests for readability
package v2

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
)

func TestGitProvider_CheckPermissions(t *testing.T) {
	user := "user"
	pass := "pass"

	config := Config{
		Username: user,
		Token:    pass,
	}
	gp := NewGitProvider(config)

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAxNTY2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IEhFQUQAbXVsdGlfYWNrIHRoaW4tcGFjayBzaWRlLWJhbmQgc2lkZS1iYW5kLTY0ayBvZnMtZGVsdGEgc2hhbGxvdyBkZWVwZW4tc2luY2UgZGVlcGVuLW5vdCBkZWVwZW4tcmVsYXRpdmUgbm8tcHJvZ3Jlc3MgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIGFsbG93LXRpcC1zaGExLWluLXdhbnQgYWxsb3ctcmVhY2hhYmxlLXNoYTEtaW4td2FudCBuby1kb25lIHN5bXJlZj1IRUFEOnJlZnMvaGVhZHMvbWFzdGVyIGZpbHRlciBvYmplY3QtZm9ybWF0PXNoYTEgYWdlbnQ9Z2l0L2dpdGh1Yi1nNzhiNDUyNDEzZThiCjAwM2ZlOGQzZmZhYjU1Mjg5NWMxOWI5ZmNmN2FhMjY0ZDI3N2NkZTMzODgxIHJlZnMvaGVhZHMvYnJhbmNoCjAwM2Y2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IHJlZnMvaGVhZHMvbWFzdGVyCjAwM2ViOGU0NzFmNThiY2JjYTYzYjA3YmRhMjBlNDI4MTkwNDA5YzJkYjQ3IHJlZnMvcHVsbC8xL2hlYWQKMDAzZTk2MzJmMDI4MzNiMmY5NjEzYWZiNWU3NTY4MjEzMmIwYjIyZTRhMzEgcmVmcy9wdWxsLzIvaGVhZAowMDNmYzM3ZjU4YTEzMGNhNTU1ZTQyZmY5NmEwNzFjYjljY2IzZjQzNzUwNCByZWZzL3B1bGwvMi9tZXJnZQowMDAw`) // nolint:lll
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bts)
		assert.NoError(t, err, "failed to write response")
	}))
	defer s.Close()

	err = gp.CheckPermissions(context.Background(), s.URL)
	require.NoError(t, err, "repo must be accessible")
}

func TestGitProvider_CheckPermissions_NoRefs(t *testing.T) {
	user := "user"
	pass := "pass"

	config := Config{
		Username: user,
		Token:    pass,
	}
	gp := NewGitProvider(config)

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAwZGUwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIGNhcGFiaWxpdGllc157fQAgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIG11bHRpX2FjayBvZnMtZGVsdGEgc2lkZS1iYW5kIHNpZGUtYmFuZC02NGsgdGhpbi1wYWNrIG5vLXByb2dyZXNzIHNoYWxsb3cgbm8tZG9uZSBhZ2VudD1KR2l0L3Y1LjkuMC4yMDIwMDkwODA1MDEtci00MS1nNWQ5MjVlY2JiCjAwMDA=`) // nolint:lll
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bts)
		assert.NoError(t, err, "failed to write response")
	}))
	defer s.Close()

	mockLogger := platform.NewLoggerMock()

	// v2 implementation returns nil for empty repos (they are technically accessible, just empty)
	// This is different from v1 which logged an error
	err = gp.CheckPermissions(ctrl.LoggerInto(context.Background(), mockLogger), s.URL)
	require.NoError(t, err, "v2 considers empty repos accessible")
}

func TestGitProvider_CreateChildBranch(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		parent   string
		child    string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "should create child branch successfully",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Create initial commit on master branch
				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test content")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial commit", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Create a parent branch and check it out so it exists as a proper reference
				err = w.Checkout(&gogit.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("parent-branch"),
					Create: true,
				})
				require.NoError(t, err)

				return dir
			},
			parent:  "parent-branch",
			child:   "child-branch",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CreateChildBranch(context.Background(), dir, tt.parent, tt.child)
			tt.wantErr(t, err)
		})
	}
}

func TestGitProvider_RemoveBranch(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		branch   string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "should remove branch successfully",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Create initial commit
				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test content")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial commit", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Create a new branch
				err = w.Checkout(&gogit.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("test-branch"),
					Create: true,
				})
				require.NoError(t, err)

				// Checkout back to master so we can delete test-branch
				err = w.Checkout(&gogit.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("master"),
				})
				require.NoError(t, err)

				return dir
			},
			branch:  "test-branch",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.RemoveBranch(context.Background(), dir, tt.branch)
			tt.wantErr(t, err)
		})
	}
}

func TestGitProvider_CommitChanges(t *testing.T) {
	tests := []struct {
		name      string
		ops       []CommitOps
		initRepo  func(t *testing.T) string
		wantErr   require.ErrorAssertionFunc
		checkRepo func(t *testing.T, dir string)
	}{
		{
			name: "should commit changes successfully",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				_, err = os.Create(path.Join(dir, "config.yaml"))
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)

				commits, err := r.CommitObjects()
				require.NoError(t, err)

				count := 0
				_ = commits.ForEach(func(*object.Commit) error {
					count++

					return nil
				})

				require.Equalf(t, 1, count, "expected 1 commits, got %d", count)
			},
		},
		{
			name: "skip commit if no changes",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)

				commits, err := r.CommitObjects()
				require.NoError(t, err)

				count := 0
				_ = commits.ForEach(func(*object.Commit) error {
					count++

					return nil
				})

				require.Equalf(t, 0, count, "expected 0 commits, got %d", count)
			},
		},
		{
			name: "should create empty commit",
			ops: []CommitOps{
				CommitAllowEmpty(),
			},
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)

				commits, err := r.CommitObjects()
				require.NoError(t, err)

				count := 0
				_ = commits.ForEach(func(*object.Commit) error {
					count++

					return nil
				})

				require.Equalf(t, 1, count, "expected 1 commits, got %d", count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.Commit(context.Background(), dir, "test commit message", tt.ops...)
			tt.wantErr(t, err)
			tt.checkRepo(t, dir)
		})
	}
}

func TestGitProvider_AddRemoteLink(t *testing.T) {
	tests := []struct {
		name      string
		remoteUrl string
		initRepo  func(t *testing.T) string
		wantErr   require.ErrorAssertionFunc
		checkRepo func(t *testing.T, dir string)
	}{
		{
			name:      "should add remote link successfully",
			remoteUrl: "git@host:32/app.git",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)

				remote, err := r.Remote("origin")
				require.NoError(t, err)

				require.Equal(t, "origin", remote.Config().Name)
				require.Len(t, remote.Config().URLs, 1)
				require.Equal(t, "git@host:32/app.git", remote.Config().URLs[0])
			},
		},
		{
			name:      "empty git dir",
			remoteUrl: "git@host:32/app.git",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()

				return dir
			},
			wantErr:   require.Error,
			checkRepo: func(t *testing.T, dir string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.AddRemoteLink(context.Background(), dir, tt.remoteUrl)
			tt.wantErr(t, err)
			tt.checkRepo(t, dir)
		})
	}
}

func TestGitProvider_CheckReference(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		from     string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "should return nil for empty reference",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			from:    "",
			wantErr: require.NoError,
		},
		{
			name: "should find existing branch reference",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Create initial commit
				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test content")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial commit", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Create and checkout a new branch
				err = w.Checkout(&gogit.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("test-branch"),
					Create: true,
				})
				require.NoError(t, err)

				return dir
			},
			from:    "test-branch",
			wantErr: require.NoError,
		},
		{
			name: "should find existing commit reference",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Create initial commit
				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test content")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				commit, err := w.Commit("initial commit", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Store the commit hash for the test
				t.Logf("Created commit with hash: %s", commit.String())

				return dir
			},
			from:    "", // Will be set dynamically
			wantErr: require.NoError,
		},
		{
			name: "should return error for non-existent reference",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			from:    "non-existent",
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			// For the commit reference test, we need to get the actual commit hash
			if tt.name == "should find existing commit reference" {
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)

				ref, err := r.Head()
				require.NoError(t, err)

				tt.from = ref.Hash().String()
				t.Logf("Using commit hash: %s", tt.from)
			}

			err := gp.CheckReference(context.Background(), dir, tt.from)
			tt.wantErr(t, err)
		})
	}
}

func TestGitProvider_getAuth(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		wantNil bool
	}{
		{
			name: "token authentication",
			config: Config{
				Token:    "test-token",
				Username: "test-user",
			},
			wantErr: false,
			wantNil: false,
		},
		{
			name: "no authentication",
			config: Config{
				Token:  "",
				SSHKey: "",
			},
			wantErr: false,
			wantNil: true,
		},
		{
			name: "invalid SSH key",
			config: Config{
				SSHKey: "invalid-key-format",
			},
			wantErr: true,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(tt.config)
			auth, err := gp.getAuth()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, auth)
			} else {
				assert.NotNil(t, auth)
			}
		})
	}
}

func TestGitProvider_getTokenAuth(t *testing.T) {
	tests := []struct {
		name         string
		gitProvider  string
		username     string
		token        string
		wantUsername string
		wantPassword string
	}{
		{
			name:         "GitHub token format",
			gitProvider:  "github",
			username:     "test-user",
			token:        "ghp_test_token",
			wantUsername: "test-user",
			wantPassword: "ghp_test_token",
		},
		{
			name:         "GitLab token format (oauth2)",
			gitProvider:  "gitlab",
			username:     "test-user",
			token:        "glpat-test-token",
			wantUsername: "oauth2",
			wantPassword: "glpat-test-token",
		},
		{
			name:         "Bitbucket token format",
			gitProvider:  "bitbucket",
			username:     "test-user",
			token:        "test-token",
			wantUsername: "test-user",
			wantPassword: "test-token",
		},
		{
			name:         "default format (unknown provider)",
			gitProvider:  "unknown",
			username:     "test-user",
			token:        "test-token",
			wantUsername: "test-user",
			wantPassword: "test-token",
		},
		{
			name:         "gerrit provider uses default",
			gitProvider:  "gerrit",
			username:     "test-user",
			token:        "test-token",
			wantUsername: "test-user",
			wantPassword: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{
				GitProvider: tt.gitProvider,
				Username:    tt.username,
				Token:       tt.token,
			})

			auth := gp.getTokenAuth()
			require.NotNil(t, auth)

			// Type assert to *githttp.BasicAuth to access Username and Password
			basicAuth, ok := auth.(*githttp.BasicAuth)
			require.True(t, ok, "auth should be *githttp.BasicAuth")
			assert.Equal(t, tt.wantUsername, basicAuth.Username)
			assert.Equal(t, tt.wantPassword, basicAuth.Password)
		})
	}
}

func TestGitProvider_NewGitProvider(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		wantSSHUser     string
		wantSSHPort     int32
		wantGitProvider string
	}{
		{
			name: "default SSH user and port",
			config: Config{
				GitProvider: "github",
			},
			wantSSHUser:     "git",
			wantSSHPort:     22,
			wantGitProvider: "github",
		},
		{
			name: "custom SSH user and port",
			config: Config{
				GitProvider: "gitlab",
				SSHUser:     "custom-user",
				SSHPort:     2222,
			},
			wantSSHUser:     "custom-user",
			wantSSHPort:     2222,
			wantGitProvider: "gitlab",
		},
		{
			name: "partial custom config (only user)",
			config: Config{
				GitProvider: "bitbucket",
				SSHUser:     "admin",
			},
			wantSSHUser:     "admin",
			wantSSHPort:     22,
			wantGitProvider: "bitbucket",
		},
		{
			name: "partial custom config (only port)",
			config: Config{
				GitProvider: "gerrit",
				SSHPort:     29418,
			},
			wantSSHUser:     "git",
			wantSSHPort:     29418,
			wantGitProvider: "gerrit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(tt.config)

			assert.Equal(t, tt.wantSSHUser, gp.config.SSHUser)
			assert.Equal(t, tt.wantSSHPort, gp.config.SSHPort)
			assert.Equal(t, tt.wantGitProvider, gp.config.GitProvider)
		})
	}
}

func TestGitProvider_Init(t *testing.T) {
	tests := []struct {
		name      string
		setupDir  func(t *testing.T) string
		wantErr   bool
		checkRepo func(t *testing.T, dir string)
	}{
		{
			name: "initialize new repository",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
			checkRepo: func(t *testing.T, dir string) {
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)
				assert.NotNil(t, r)
			},
		},
		{
			name: "initialize already initialized repository",
			setupDir: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			wantErr: true,
			checkRepo: func(t *testing.T, dir string) {
				// Repository should still be valid even though init failed
				r, err := gogit.PlainOpen(dir)
				require.NoError(t, err)
				assert.NotNil(t, r)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.setupDir(t)

			err := gp.Init(context.Background(), dir)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.checkRepo(t, dir)
		})
	}
}

func TestGitProvider_GetCurrentBranchName(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		wantBranch string
		wantErr    bool
	}{
		{
			name: "get current branch name",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Create initial commit
				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			wantBranch: "master",
			wantErr:    false,
		},
		{
			name: "repository not found",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			wantBranch: "",
			wantErr:    true,
		},
		{
			name: "get branch after checkout",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Create and checkout a new branch
				err = w.Checkout(&gogit.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("feature-branch"),
					Create: true,
				})
				require.NoError(t, err)

				return dir
			},
			wantBranch: "feature-branch",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			branch, err := gp.GetCurrentBranchName(context.Background(), dir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantBranch, branch)
		})
	}
}

func TestGitProvider_CommitExists(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) (string, string)
		commitHash string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "commit exists",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				hash, err := w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir, hash.String()
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "commit does not exist",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Return a valid-looking but non-existent commit hash
				return dir, "0000000000000000000000000000000000000000"
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "invalid hash - treated as not found",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				// Create a commit so the repo has valid objects
				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Invalid hash format is treated as not found, not an error
				return dir, "invalid-hash"
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "repository error",
			initRepo: func(t *testing.T) (string, string) {
				// Return a non-git directory
				return t.TempDir(), "0000000000000000000000000000000000000000"
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir, hash := tt.initRepo(t)

			exists, err := gp.CommitExists(context.Background(), dir, hash)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}

func TestBranchExists(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "branch exists",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				// Create test-branch
				err = w.Checkout(&gogit.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("test-branch"),
					Create: true,
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "test-branch",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "branch does not exist",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "non-existent-branch",
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.initRepo(t)
			r, err := gogit.PlainOpen(dir)
			require.NoError(t, err)

			branches, err := r.Branches()
			require.NoError(t, err)

			exists, err := branchExists(tt.branchName, branches)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}

func TestResolveReference(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) (string, string)
		ref      string
		wantErr  bool
	}{
		{
			name: "empty ref uses HEAD",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir, ""
			},
			ref:     "",
			wantErr: false,
		},
		{
			name: "branch reference resolution",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir, "master"
			},
			ref:     "master",
			wantErr: false,
		},
		{
			name: "commit hash resolution",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				hash, err := w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir, hash.String()
			},
			wantErr: false,
		},
		{
			name: "invalid reference",
			initRepo: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir, "invalid-ref-12345"
			},
			ref:     "invalid-ref-12345",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, ref := tt.initRepo(t)
			r, err := gogit.PlainOpen(dir)
			require.NoError(t, err)

			// Use the ref from initRepo if one was provided
			testRef := tt.ref
			if testRef == "" && ref != "" {
				testRef = ref
			}

			hash, err := resolveReference(r, testRef)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, hash)
		})
	}
}

func TestGitProvider_Checkout_LocalMode(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		remote     bool
		wantErr    bool
	}{
		{
			name: "checkout local branch (remote=false)",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				// Create initial commit
				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "test-branch",
			remote:     false,
			wantErr:    false,
		},
		{
			name: "checkout on non-existent directory",
			initRepo: func(t *testing.T) string {
				return t.TempDir() // Empty directory, not a git repo
			},
			branchName: "test-branch",
			remote:     false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.Checkout(context.Background(), dir, tt.branchName, tt.remote)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify the branch was checked out
			r, err := gogit.PlainOpen(dir)
			require.NoError(t, err)

			head, err := r.Head()
			require.NoError(t, err)

			assert.Equal(t, tt.branchName, head.Name().Short())
		})
	}
}

func TestGitProvider_CreateRemoteBranch_Errors(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		fromRef    string
		wantErr    bool
	}{
		{
			name: "repository not found",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			branchName: "new-branch",
			fromRef:    "master",
			wantErr:    true,
		},
		{
			name: "invalid fromRef",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "new-branch",
			fromRef:    "non-existent-ref",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CreateRemoteBranch(context.Background(), dir, tt.branchName, tt.fromRef)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_CreateRemoteTag_Errors(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		tagName    string
		branchName string
		wantErr    bool
	}{
		{
			name: "repository not found",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			tagName:    "v1.0.0",
			branchName: "master",
			wantErr:    true,
		},
		{
			name: "branch not found",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			tagName:    "v1.0.0",
			branchName: "non-existent-branch",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CreateRemoteTag(context.Background(), dir, tt.tagName, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_Clone_Errors(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		wantErr bool
	}{
		{
			name:    "clone with empty URL",
			repoURL: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			destination := t.TempDir()

			err := gp.Clone(context.Background(), tt.repoURL, destination)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_Commit_ErrorCases(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		message  string
		wantErr  bool
	}{
		{
			name: "commit with invalid directory",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			message: "test commit",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.Commit(context.Background(), dir, tt.message)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_Push_ErrorCases(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		refspec  string
		wantErr  bool
	}{
		{
			name: "push with invalid directory",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			refspec: RefSpecPushAllBranches,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.Push(context.Background(), dir, tt.refspec)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_CheckoutRemoteBranch_Errors(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		wantErr    bool
	}{
		{
			name: "invalid directory",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			branchName: "feature-branch",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CheckoutRemoteBranch(context.Background(), dir, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_CreateChildBranch_Errors(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		parentName string
		childName  string
		wantErr    bool
	}{
		{
			name: "invalid directory",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			parentName: "parent",
			childName:  "child",
			wantErr:    true,
		},
		{
			name: "non-existent parent",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			parentName: "non-existent-parent",
			childName:  "child",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CreateChildBranch(context.Background(), dir, tt.parentName, tt.childName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_RemoveBranch_Errors(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		wantErr    bool
	}{
		{
			name: "invalid directory",
			initRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			branchName: "test-branch",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.RemoveBranch(context.Background(), dir, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_Clone_Basic(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func(t *testing.T) (string, string)
		wantErr  bool
	}{
		{
			name: "invalid URL",
			setupEnv: func(t *testing.T) (string, string) {
				targetDir := t.TempDir()
				return targetDir, "invalid-url://invalid"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			targetDir, repoURL := tt.setupEnv(t)

			err := gp.Clone(context.Background(), repoURL, targetDir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify the clone succeeded
			_, err = gogit.PlainOpen(targetDir)
			require.NoError(t, err)
		})
	}
}

func TestGitProvider_Push_NoRemote(t *testing.T) {
	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		refspec  string
		wantErr  bool
	}{
		{
			name: "push without remote configured",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			refspec: RefSpecPushAllBranches,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.Push(context.Background(), dir, tt.refspec)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_Fetch_NoRemote(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		wantErr    bool
	}{
		{
			name: "fetch without remote configured",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "",
			wantErr:    true,
		},
		{
			name: "fetch specific branch without remote",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "master",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.Fetch(context.Background(), dir, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_CheckoutRemoteBranch_NoRemote(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		branchName string
		wantErr    bool
	}{
		{
			name: "checkout remote branch without remote",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			branchName: "feature-branch",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CheckoutRemoteBranch(context.Background(), dir, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGitProvider_CreateRemoteTag_NoRemote(t *testing.T) {
	tests := []struct {
		name       string
		initRepo   func(t *testing.T) string
		tagName    string
		branchName string
		wantErr    bool
	}{
		{
			name: "create tag without remote",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				r, err := gogit.PlainInit(dir, false)
				require.NoError(t, err)

				w, err := r.Worktree()
				require.NoError(t, err)

				f, err := os.Create(path.Join(dir, "test.txt"))
				require.NoError(t, err)
				_, err = f.WriteString("test")
				require.NoError(t, err)
				require.NoError(t, f.Close())

				_, err = w.Add("test.txt")
				require.NoError(t, err)

				_, err = w.Commit("initial", &gogit.CommitOptions{
					Author: &object.Signature{
						Name:  "test",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)

				return dir
			},
			tagName:    "v1.0.0",
			branchName: "master",
			wantErr:    true, // Will fail on push
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGitProvider(Config{})
			dir := tt.initRepo(t)

			err := gp.CreateRemoteTag(context.Background(), dir, tt.tagName, tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
