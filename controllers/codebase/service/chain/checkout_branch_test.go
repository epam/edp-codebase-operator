package chain

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	testify "github.com/stretchr/testify/mock"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitServerMocks "github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	fakeNamespace = "fake_namespace"
	fakeName      = "fake-name"
)

func TestGetRepositoryCredentialsIfExists_ShouldPass(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()
	u, p, err := GetRepositoryCredentialsIfExists(c, fakeCl)
	assert.Equal(t, u, util.GetStringP("user"))
	assert.Equal(t, p, util.GetStringP("pass"))
	assert.NoError(t, err)
}

func TestGetRepositoryCredentialsIfExists_ShouldFail(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	_, _, err := GetRepositoryCredentialsIfExists(c, fakeCl)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to get secret repository-codebase-fake-name-temp") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnGetSecret(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	mGit := gitServerMocks.NewMockGit(t)

	err := CheckoutBranch("repo", "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to get secret repository-codebase-fake-name-temp") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnCheckPermission(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()

	mGit := gitServerMocks.NewMockGit(t)
	mGit.On("CheckPermissions", testify.Anything, "repo", util.GetStringP("user"), util.GetStringP("pass")).Return(false)

	err := CheckoutBranch("repo", "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "user user cannot get access to the repository repo") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnGetCurrentBranchName(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()

	mGit := gitServerMocks.NewMockGit(t)
	mGit.On("CheckPermissions", testify.Anything, "repo", util.GetStringP("user"), util.GetStringP("pass")).Return(true)
	mGit.On("GetCurrentBranchName", "project-path").Return("", errors.New("FATAL:FAILED"))

	err := CheckoutBranch("repo", "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "FATAL:FAILED") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnCheckout(t *testing.T) {
	repo := "repo"
	u := "user"
	p := "pass"
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			Strategy: codebaseApi.Clone,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()

	mGit := gitServerMocks.NewMockGit(t)
	mGit.On("CheckPermissions", testify.Anything, "repo", &u, &p).Return(true)
	mGit.On("GetCurrentBranchName", "project-path").Return("some-other-branch", nil)
	mGit.On("Checkout", &u, &p, "project-path", "branch", true).Return(errors.New("FATAL:FAILED"))

	err := CheckoutBranch(repo, "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "FATAL:FAILED") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldPassForCloneStrategy(t *testing.T) {
	repo := "repo"
	u := "user"
	p := "pass"
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer: "git",
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			Strategy: codebaseApi.Import,
		},
	}
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "git",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s, ssh)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c, gs, ssh).Build()

	mGit := gitServerMocks.NewMockGit(t)
	mGit.On("CheckPermissions", testify.Anything, "repo", &u, &p).Return(true)
	mGit.On("GetCurrentBranchName", "project-path").Return("some-other-branch", nil)
	mGit.On("CheckoutRemoteBranchBySSH", "fake", fakeName, "project-path", "branch").Return(nil)

	err := CheckoutBranch(repo, "project-path", "branch", mGit, c, fakeCl)
	assert.NoError(t, err)
}
