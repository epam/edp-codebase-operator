package gitserver

import (
	"encoding/base64"
	"errors"
	"os"
	"path"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
)

func TestGitProvider_CheckPermissions(t *testing.T) {
	gp := GitProvider{}
	user := "user"
	pass := "pass"

	httpmock.Reset()
	httpmock.Activate()

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAxNTY2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IEhFQUQAbXVsdGlfYWNrIHRoaW4tcGFjayBzaWRlLWJhbmQgc2lkZS1iYW5kLTY0ayBvZnMtZGVsdGEgc2hhbGxvdyBkZWVwZW4tc2luY2UgZGVlcGVuLW5vdCBkZWVwZW4tcmVsYXRpdmUgbm8tcHJvZ3Jlc3MgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIGFsbG93LXRpcC1zaGExLWluLXdhbnQgYWxsb3ctcmVhY2hhYmxlLXNoYTEtaW4td2FudCBuby1kb25lIHN5bXJlZj1IRUFEOnJlZnMvaGVhZHMvbWFzdGVyIGZpbHRlciBvYmplY3QtZm9ybWF0PXNoYTEgYWdlbnQ9Z2l0L2dpdGh1Yi1nNzhiNDUyNDEzZThiCjAwM2ZlOGQzZmZhYjU1Mjg5NWMxOWI5ZmNmN2FhMjY0ZDI3N2NkZTMzODgxIHJlZnMvaGVhZHMvYnJhbmNoCjAwM2Y2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IHJlZnMvaGVhZHMvbWFzdGVyCjAwM2ViOGU0NzFmNThiY2JjYTYzYjA3YmRhMjBlNDI4MTkwNDA5YzJkYjQ3IHJlZnMvcHVsbC8xL2hlYWQKMDAzZTk2MzJmMDI4MzNiMmY5NjEzYWZiNWU3NTY4MjEzMmIwYjIyZTRhMzEgcmVmcy9wdWxsLzIvaGVhZAowMDNmYzM3ZjU4YTEzMGNhNTU1ZTQyZmY5NmEwNzFjYjljY2IzZjQzNzUwNCByZWZzL3B1bGwvMi9tZXJnZQowMDAw`)
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder("GET", "http://repo.git/info/refs?service=git-upload-pack",
		httpmock.NewBytesResponder(200, bts))

	if !gp.CheckPermissions("http://repo.git", &user, &pass) {
		t.Fatal("repo must be accessible")
	}
}

func TestGitProvider_CheckPermissions_NoRefs(t *testing.T) {
	gp := GitProvider{}
	user := "user"
	pass := "pass"

	httpmock.Reset()
	httpmock.Activate()

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAwZGUwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIGNhcGFiaWxpdGllc157fQAgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIG11bHRpX2FjayBvZnMtZGVsdGEgc2lkZS1iYW5kIHNpZGUtYmFuZC02NGsgdGhpbi1wYWNrIG5vLXByb2dyZXNzIHNoYWxsb3cgbm8tZG9uZSBhZ2VudD1KR2l0L3Y1LjkuMC4yMDIwMDkwODA1MDEtci00MS1nNWQ5MjVlY2JiCjAwMDA=`)
	if err != nil {
		t.Fatal(err)
	}

	mockLogger := platform.NewLoggerMock()
	loggerSink, ok := mockLogger.GetSink().(*platform.LoggerMock)
	require.True(t, ok)

	log = mockLogger

	httpmock.RegisterResponder("GET", "http://repo.git/info/refs?service=git-upload-pack",
		httpmock.NewBytesResponder(200, bts))

	if gp.CheckPermissions("http://repo.git", &user, &pass) {
		t.Fatal("repo must be not accessible")
	}

	lastErr := loggerSink.LastError()
	if lastErr == nil {
		t.Fatal("no error logged")
	}

	if lastErr.Error() != "there are not refs in repository" {
		t.Fatalf("wrong error returned: %s", lastErr.Error())
	}
}

func TestInitAuth(t *testing.T) {
	dir, err := initAuth("foo", "bar")
	assert.NoError(t, err)
	assert.Contains(t, dir, "sshkey")
}

func TestGitProvider_CreateChildBranch(t *testing.T) {
	cm := mocks.CommandMock{}
	gp := GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)

	err := gp.CreateChildBranch("dir", "br1", "br2")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := mocks.CommandMock{}
	gp = GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
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
	gp := GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)

	err := gp.RemoveBranch("dir", "br1")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := mocks.CommandMock{}
	gp = GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
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
	gp := GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)

	err := gp.RenameBranch("dir", "br1", "br2")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := mocks.CommandMock{}
	gp = GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()

	err = gp.RenameBranch("dir", "br1", "br2")
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to checkout branch, err: : fatal")
}

func Test_publicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "success",
			key:     testKey,
			wantErr: assert.NoError,
		},
		{
			name:    "success",
			key:     "invalid-key",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := publicKey(tt.key)
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := initAuth(tt.key, "user")
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
		initRepo  func(t *testing.T) string
		wantErr   require.ErrorAssertionFunc
		checkRepo func(t *testing.T, dir string)
	}{
		{
			name: "should commit changes successfully",
			initRepo: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := git.PlainInit(dir, false)
				require.NoError(t, err)

				_, err = os.Create(path.Join(dir, "config.yaml"))
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := git.PlainOpen(dir)
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
				_, err := git.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := git.PlainOpen(dir)
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gp := &GitProvider{}
			dir := tt.initRepo(t)

			err := gp.CommitChanges(dir, "test commit message")
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
				_, err := git.PlainInit(dir, false)
				require.NoError(t, err)

				return dir
			},
			wantErr: require.NoError,
			checkRepo: func(t *testing.T, dir string) {
				r, err := git.PlainOpen(dir)
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

			gp := &GitProvider{}
			dir := tt.initRepo(t)

			err := gp.AddRemoteLink(dir, tt.remoteUrl)
			tt.wantErr(t, err)
			tt.checkRepo(t, dir)
		})
	}
}
