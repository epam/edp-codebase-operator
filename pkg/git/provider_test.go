// nolint:dupl // Duplicate test setup is acceptable in tests for readability
package v2

import (
	"context"
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

	s := uploadPackServer(t)

	err := gp.CheckPermissions(context.Background(), s.URL)
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

	s := emptyUploadPackServer(t)

	mockLogger := platform.NewLoggerMock()

	// v2 implementation returns nil for empty repos (they are technically accessible, just empty)
	// This is different from v1 which logged an error
	err := gp.CheckPermissions(ctrl.LoggerInto(context.Background(), mockLogger), s.URL)
	require.NoError(t, err, "v2 considers empty repos accessible")
}

func TestGitProvider_ListRemoteBranches(t *testing.T) {
	config := Config{
		Username: "user",
		Token:    "pass",
	}
	gp := NewGitProvider(config)

	// Advertisement contains refs/heads/branch, refs/heads/master plus
	// pull-request refs that must be filtered out.
	s := uploadPackServer(t)

	branches, err := gp.ListRemoteBranches(context.Background(), s.URL)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"branch", "master"}, branches)
}

func TestGitProvider_ListRemoteBranches_EmptyRepo(t *testing.T) {
	config := Config{
		Username: "user",
		Token:    "pass",
	}
	gp := NewGitProvider(config)

	s := emptyUploadPackServer(t)

	branches, err := gp.ListRemoteBranches(context.Background(), s.URL)
	require.NoError(t, err)
	assert.Empty(t, branches)
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

// Host key lookups are keyed on the port actually dialled, which the repository
// URL determines: ssh:// carries a port, the scp-style form cannot.
func TestSshTarget(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    string
	}{
		{
			name:    "ssh url carries its port",
			repoURL: "ssh://gerrit.example.com:29418/my-project",
			want:    "gerrit.example.com:29418",
		},
		{
			name:    "scp-style url has no port and defaults to 22",
			repoURL: "git@github.com:owner/repo.git",
			want:    "github.com:22",
		},
		{
			name:    "ssh url with a non-standard port",
			repoURL: "ssh://git@git.example.com:2222/owner/repo.git",
			want:    "git.example.com:2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sshTarget(tt.repoURL))
		})
	}
}
