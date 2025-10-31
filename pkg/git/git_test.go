package git_test

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
)

func TestGitProvider_CheckPermissions(t *testing.T) {
	user := "user"
	pass := "pass"

	config := gitproviderv2.Config{
		Username: user,
		Token:    pass,
	}
	gp := gitproviderv2.NewGitProvider(config)

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAxNTY2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IEhFQUQAbXVsdGlfYWNrIHRoaW4tcGFjayBzaWRlLWJhbmQgc2lkZS1iYW5kLTY0ayBvZnMtZGVsdGEgc2hhbGxvdyBkZWVwZW4tc2luY2UgZGVlcGVuLW5vdCBkZWVwZW4tcmVsYXRpdmUgbm8tcHJvZ3Jlc3MgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIGFsbG93LXRpcC1zaGExLWluLXdhbnQgYWxsb3ctcmVhY2hhYmxlLXNoYTEtaW4td2FudCBuby1kb25lIHN5bXJlZj1IRUFEOnJlZnMvaGVhZHMvbWFzdGVyIGZpbHRlciBvYmplY3QtZm9ybWF0PXNoYTEgYWdlbnQ9Z2l0L2dpdGh1Yi1nNzhiNDUyNDEzZThiCjAwM2ZlOGQzZmZhYjU1Mjg5NWMxOWI5ZmNmN2FhMjY0ZDI3N2NkZTMzODgxIHJlZnMvaGVhZHMvYnJhbmNoCjAwM2Y2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IHJlZnMvaGVhZHMvbWFzdGVyCjAwM2ViOGU0NzFmNThiY2JjYTYzYjA3YmRhMjBlNDI4MTkwNDA5YzJkYjQ3IHJlZnMvcHVsbC8xL2hlYWQKMDAzZTk2MzJmMDI4MzNiMmY5NjEzYWZiNWU3NTY4MjEzMmIwYjIyZTRhMzEgcmVmcy9wdWxsLzIvaGVhZAowMDNmYzM3ZjU4YTEzMGNhNTU1ZTQyZmY5NmEwNzFjYjljY2IzZjQzNzUwNCByZWZzL3B1bGwvMi9tZXJnZQowMDAw`)
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

	config := gitproviderv2.Config{
		Username: user,
		Token:    pass,
	}
	gp := gitproviderv2.NewGitProvider(config)

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAwZGUwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIGNhcGFiaWxpdGllc157fQAgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIG11bHRpX2FjayBvZnMtZGVsdGEgc2lkZS1iYW5kIHNpZGUtYmFuZC02NGsgdGhpbi1wYWNrIG5vLXByb2dyZXNzIHNoYWxsb3cgbm8tZG9uZSBhZ2VudD1KR2l0L3Y1LjkuMC4yMDIwMDkwODA1MDEtci00MS1nNWQ5MjVlY2JiCjAwMDA=`)
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

func TestInitAuth(t *testing.T) {
	dir, err := git.InitAuth("foo", "bar")
	assert.NoError(t, err)
	assert.Contains(t, dir, "sshkey")
}

func TestGitProvider_CreateChildBranch(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			gp := gitproviderv2.NewGitProvider(gitproviderv2.Config{})
			dir := tt.initRepo(t)

			err := gp.CreateChildBranch(context.Background(), dir, tt.parent, tt.child)
			tt.wantErr(t, err)
		})
	}
}

func TestGitProvider_RemoveBranch(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			gp := gitproviderv2.NewGitProvider(gitproviderv2.Config{})
			dir := tt.initRepo(t)

			err := gp.RemoveBranch(context.Background(), dir, tt.branch)
			tt.wantErr(t, err)
		})
	}
}

func TestGitProvider_RenameBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initRepo func(t *testing.T) string
		oldName  string
		newName  string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "should rename branch successfully",
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

				return dir
			},
			oldName: "master",
			newName: "main",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gp := gitproviderv2.NewGitProvider(gitproviderv2.Config{})
			dir := tt.initRepo(t)

			err := gp.RenameBranch(context.Background(), dir, tt.oldName, tt.newName)
			tt.wantErr(t, err)
		})
	}
}

func Test_initAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success without empty line in the end",
			key: `-----KEY-----
some-key
-----END-----`,
			want: `-----KEY-----
some-key
-----END-----
`,
			wantErr: require.NoError,
		},
		{
			name: "success with empty line in the end",
			key: `-----KEY-----
some-key
-----END-----
`,
			want: `-----KEY-----
some-key
-----END-----

`,
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := git.InitAuth(tt.key, "user")
			tt.wantErr(t, err)

			gotKey, err := os.ReadFile(got)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(gotKey))
		})
	}
}

func TestGitProvider_CommitChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ops       []gitproviderv2.CommitOps
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
			ops: []gitproviderv2.CommitOps{
				gitproviderv2.CommitAllowEmpty(),
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
			t.Parallel()

			gp := gitproviderv2.NewGitProvider(gitproviderv2.Config{})
			dir := tt.initRepo(t)

			err := gp.Commit(context.Background(), dir, "test commit message", tt.ops...)
			tt.wantErr(t, err)
			tt.checkRepo(t, dir)
		})
	}
}

func TestGitProvider_AddRemoteLink(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			gp := gitproviderv2.NewGitProvider(gitproviderv2.Config{})
			dir := tt.initRepo(t)

			err := gp.AddRemoteLink(context.Background(), dir, tt.remoteUrl)
			tt.wantErr(t, err)
			tt.checkRepo(t, dir)
		})
	}
}

func TestGitProvider_CheckReference(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			gp := gitproviderv2.NewGitProvider(gitproviderv2.Config{})
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
