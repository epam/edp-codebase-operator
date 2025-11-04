package chain

import (
	"context"
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
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	gitServerMocks "github.com/epam/edp-codebase-operator/v2/pkg/git/v2/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	fakeNamespace = "fake_namespace"
	fakeName      = "fake-name"
)

func TestCheckoutBranch_ShouldFailOnGetCurrentBranchName(t *testing.T) {
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
			Strategy: codebaseApi.Clone,
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
			util.PrivateSShKeyName: []byte("fake-ssh-key"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s, ssh)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c, gs, ssh).Build()

	mGit := gitServerMocks.NewMockGit(t)
	mGit.On("GetCurrentBranchName", testify.Anything, "project-path").Return("", errors.New("FATAL:FAILED"))

	err := CheckoutBranch(context.Background(), "branch", &GitRepositoryContext{
		GitServer:       gs,
		GitServerSecret: ssh,
		PrivateSSHKey:   "fake-ssh-key",
		UserName:        "user",
		Token:           "pass",
		RepoGitUrl:      "repo",
		WorkDir:         "project-path",
	}, c, fakeCl, func(config gitproviderv2.Config) gitproviderv2.Git {
		return mGit
	})
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "FATAL:FAILED") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnCheckout(t *testing.T) {
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
			Strategy: codebaseApi.Clone,
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
			"username": []byte("user1"),
			"password": []byte("pass1"),
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake-ssh-key"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s, ssh)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c, gs, ssh).Build()

	mGit := gitServerMocks.NewMockGit(t)
	mGit.On("GetCurrentBranchName", testify.Anything, "project-path").Return("some-other-branch", nil)
	mGit.On("Checkout", testify.Anything, "project-path", "branch", true).Return(errors.New("FATAL:FAILED"))

	err := CheckoutBranch(context.Background(), "branch", &GitRepositoryContext{
		GitServer:       gs,
		GitServerSecret: ssh,
		PrivateSSHKey:   "fake-ssh-key",
		UserName:        "user1",
		Token:           "pass1",
		RepoGitUrl:      "repo",
		WorkDir:         "project-path",
	}, c, fakeCl, func(config gitproviderv2.Config) gitproviderv2.Git {
		return mGit
	})
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "FATAL:FAILED") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldPassForCloneStrategy(t *testing.T) {
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
			CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
				SecretRef: coreV1.LocalObjectReference{
					Name: "repository-creds",
				},
			},
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
			Name:      "repository-creds",
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
	mGit.On("GetCurrentBranchName", testify.Anything, "project-path").Return("some-other-branch", nil)
	mGit.On("CheckoutRemoteBranch", testify.Anything, "project-path", "branch").Return(nil)

	err := CheckoutBranch(context.Background(), "branch", &GitRepositoryContext{
		GitServer:       gs,
		GitServerSecret: ssh,
		PrivateSSHKey:   "fake",
		UserName:        "user",
		Token:           "pass",
		RepoGitUrl:      "repo",
		WorkDir:         "project-path",
	}, c, fakeCl, func(config gitproviderv2.Config) gitproviderv2.Git {
		return mGit
	})
	assert.NoError(t, err)
}
