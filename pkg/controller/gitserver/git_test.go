package gitserver

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/platform"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGitProvider_CheckPermissions(t *testing.T) {
	gp := GitProvider{}
	var (
		user = "user"
		pass = "pass"
	)
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
	var (
		user = "user"
		pass = "pass"
	)
	httpmock.Reset()
	httpmock.Activate()

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAwZGUwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIGNhcGFiaWxpdGllc157fQAgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIG11bHRpX2FjayBvZnMtZGVsdGEgc2lkZS1iYW5kIHNpZGUtYmFuZC02NGsgdGhpbi1wYWNrIG5vLXByb2dyZXNzIHNoYWxsb3cgbm8tZG9uZSBhZ2VudD1KR2l0L3Y1LjkuMC4yMDIwMDkwODA1MDEtci00MS1nNWQ5MjVlY2JiCjAwMDA=`)
	if err != nil {
		t.Fatal(err)
	}

	mockLogger := platform.LoggerMock{}
	log = &mockLogger

	httpmock.RegisterResponder("GET", "http://repo.git/info/refs?service=git-upload-pack",
		httpmock.NewBytesResponder(200, bts))
	if gp.CheckPermissions("http://repo.git", &user, &pass) {
		t.Fatal("repo must be not accessible")
	}

	lastErr := mockLogger.LastError()
	if lastErr == nil {
		t.Fatal("no error logged")
	}

	if lastErr.Error() != "there are not refs in repository" {
		t.Fatalf("wrong error returned: %s", lastErr.Error())
	}
}

func TestInitAuth(t *testing.T) {
	path, err := initAuth("foo", "bar")
	assert.NoError(t, err)
	assert.Contains(t, path, "sshkey")
}

func TestGitProvider_CreateChildBranch(t *testing.T) {
	cm := commandMock{}
	gp := GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)
	err := gp.CreateChildBranch("dir", "br1", "br2")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := commandMock{}
	gp = GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()
	err = gp.CreateChildBranch("dir", "br1", "br2")
	assert.Error(t, err)
	assert.EqualError(t, err, "unable to checkout branch, err: : fatal")
}

func TestGitProvider_RemoveBranch(t *testing.T) {
	cm := commandMock{}
	gp := GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)
	err := gp.RemoveBranch("dir", "br1")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := commandMock{}
	gp = GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()
	err = gp.RemoveBranch("dir", "br1")
	assert.Error(t, err)
	assert.EqualError(t, err, "unable to remove branch, err: : fatal")
}

func TestGitProvider_RenameBranch(t *testing.T) {
	cm := commandMock{}
	gp := GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cm
		},
	}

	cm.On("CombinedOutput").Return([]byte("t"), nil)
	err := gp.RenameBranch("dir", "br1", "br2")
	assert.NoError(t, err)
	cm.AssertExpectations(t)

	cmError := commandMock{}
	gp = GitProvider{
		commandBuilder: func(cmd string, params ...string) Command {
			return &cmError
		},
	}

	cmError.On("CombinedOutput").Return([]byte("t"), errors.New("fatal")).Once()
	err = gp.RenameBranch("dir", "br1", "br2")
	assert.Error(t, err)
	assert.EqualError(t, err, "unable to checkout branch, err: : fatal")
}
