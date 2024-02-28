package git_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
)

func TestGitProvider_CheckPermissions(t *testing.T) {
	gp := git.GitProvider{}
	user := "user"
	pass := "pass"

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAxNTY2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IEhFQUQAbXVsdGlfYWNrIHRoaW4tcGFjayBzaWRlLWJhbmQgc2lkZS1iYW5kLTY0ayBvZnMtZGVsdGEgc2hhbGxvdyBkZWVwZW4tc2luY2UgZGVlcGVuLW5vdCBkZWVwZW4tcmVsYXRpdmUgbm8tcHJvZ3Jlc3MgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIGFsbG93LXRpcC1zaGExLWluLXdhbnQgYWxsb3ctcmVhY2hhYmxlLXNoYTEtaW4td2FudCBuby1kb25lIHN5bXJlZj1IRUFEOnJlZnMvaGVhZHMvbWFzdGVyIGZpbHRlciBvYmplY3QtZm9ybWF0PXNoYTEgYWdlbnQ9Z2l0L2dpdGh1Yi1nNzhiNDUyNDEzZThiCjAwM2ZlOGQzZmZhYjU1Mjg5NWMxOWI5ZmNmN2FhMjY0ZDI3N2NkZTMzODgxIHJlZnMvaGVhZHMvYnJhbmNoCjAwM2Y2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IHJlZnMvaGVhZHMvbWFzdGVyCjAwM2ViOGU0NzFmNThiY2JjYTYzYjA3YmRhMjBlNDI4MTkwNDA5YzJkYjQ3IHJlZnMvcHVsbC8xL2hlYWQKMDAzZTk2MzJmMDI4MzNiMmY5NjEzYWZiNWU3NTY4MjEzMmIwYjIyZTRhMzEgcmVmcy9wdWxsLzIvaGVhZAowMDNmYzM3ZjU4YTEzMGNhNTU1ZTQyZmY5NmEwNzFjYjljY2IzZjQzNzUwNCByZWZzL3B1bGwvMi9tZXJnZQowMDAw`)
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bts)
		assert.NoError(t, err, "failed to write response")
	}))
	defer s.Close()

	require.True(t, gp.CheckPermissions(context.Background(), s.URL, &user, &pass), "repo must be accessible")
}

func TestGitProvider_CheckPermissions_NoRefs(t *testing.T) {
	gp := git.GitProvider{}
	user := "user"
	pass := "pass"

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAwZGUwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIGNhcGFiaWxpdGllc157fQAgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIG11bHRpX2FjayBvZnMtZGVsdGEgc2lkZS1iYW5kIHNpZGUtYmFuZC02NGsgdGhpbi1wYWNrIG5vLXByb2dyZXNzIHNoYWxsb3cgbm8tZG9uZSBhZ2VudD1KR2l0L3Y1LjkuMC4yMDIwMDkwODA1MDEtci00MS1nNWQ5MjVlY2JiCjAwMDA=`)
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bts)
		assert.NoError(t, err, "failed to write response")
	}))
	defer s.Close()

	mockLogger := platform.NewLoggerMock()
	loggerSink, ok := mockLogger.GetSink().(*platform.LoggerMock)
	require.True(t, ok)

	accessible := gp.CheckPermissions(ctrl.LoggerInto(context.Background(), mockLogger), s.URL, &user, &pass)
	require.False(t, accessible, "repo must not be accessible")
	require.Error(t, loggerSink.LastError())
	require.Contains(t, loggerSink.LastError().Error(), "remote repository is empty")
}

func TestInitAuth(t *testing.T) {
	dir, err := git.InitAuth("foo", "bar")
	assert.NoError(t, err)
	assert.Contains(t, dir, "sshkey")
}

func TestGitProvider_CreateChildBranch(t *testing.T) {
	cm := mocks.CommandMock{}
	gp := git.GitProvider{
		CommandBuilder: func(cmd string, params ...string) git.Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)

	err := gp.CreateChildBranch("dir", "br1", "br2")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := mocks.CommandMock{}
	gp = git.GitProvider{
		CommandBuilder: func(cmd string, params ...string) git.Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()

	err = gp.CreateChildBranch("dir", "br1", "br2")
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to checkout branch, err: : fatal")
}

func TestGitProvider_RemoveBranch(t *testing.T) {
	cm := mocks.CommandMock{}
	gp := git.GitProvider{
		CommandBuilder: func(cmd string, params ...string) git.Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)

	err := gp.RemoveBranch("dir", "br1")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := mocks.CommandMock{}
	gp = git.GitProvider{
		CommandBuilder: func(cmd string, params ...string) git.Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()

	err = gp.RemoveBranch("dir", "br1")
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to remove branch, err: : fatal")
}

func TestGitProvider_RenameBranch(t *testing.T) {
	cm := mocks.CommandMock{}
	gp := git.GitProvider{
		CommandBuilder: func(cmd string, params ...string) git.Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)

	err := gp.RenameBranch("dir", "br1", "br2")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := mocks.CommandMock{}
	gp = git.GitProvider{
		CommandBuilder: func(cmd string, params ...string) git.Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()

	err = gp.RenameBranch("dir", "br1", "br2")
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to checkout branch, err: : fatal")
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
		tt := tt

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
		ops       []git.CommitOps
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
			ops: []git.CommitOps{
				git.CommitAllowEmpty(),
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gp := &git.GitProvider{}
			dir := tt.initRepo(t)

			err := gp.CommitChanges(dir, "test commit message", tt.ops...)
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gp := &git.GitProvider{}
			dir := tt.initRepo(t)

			err := gp.AddRemoteLink(dir, tt.remoteUrl)
			tt.wantErr(t, err)
			tt.checkRepo(t, dir)
		})
	}
}
