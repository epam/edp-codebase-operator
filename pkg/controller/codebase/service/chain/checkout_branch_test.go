package chain

import (
	"errors"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	mockgit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetRepositoryCredentialsIfExists_ShouldPass(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()
	u, p, err := GetRepositoryCredentialsIfExists(c, fakeCl)
	assert.Equal(t, u, util.GetStringP("user"))
	assert.Equal(t, p, util.GetStringP("pass"))
	assert.NoError(t, err)
}

func TestGetRepositoryCredentialsIfExists_ShouldFail(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	_, _, err := GetRepositoryCredentialsIfExists(c, fakeCl)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "Unable to get secret repository-codebase-fake-name-temp") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnGetSecret(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	mGit := new(mockgit.MockGit)

	err := CheckoutBranch(util.GetStringP("repo"), "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "Unable to get secret repository-codebase-fake-name-temp") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnCheckPermission(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()

	mGit := new(mockgit.MockGit)
	mGit.On("CheckPermissions", "repo", util.GetStringP("user"), util.GetStringP("pass")).Return(false)

	err := CheckoutBranch(util.GetStringP("repo"), "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "user user cannot get access to the repository repo") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnGetCurrentBranchName(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()

	mGit := new(mockgit.MockGit)
	mGit.On("CheckPermissions", "repo", util.GetStringP("user"), util.GetStringP("pass")).Return(true)
	mGit.On("GetCurrentBranchName", "project-path").Return("", errors.New("FATAL:FAILED"))

	err := CheckoutBranch(util.GetStringP("repo"), "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "FATAL:FAILED") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldFailOnCheckout(t *testing.T) {
	var (
		repo = "repo"
		u    = "user"
		p    = "pass"
	)
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
			Strategy: v1alpha1.Clone,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c).Build()

	mGit := new(mockgit.MockGit)
	mGit.On("CheckPermissions", "repo", &u, &p).Return(true)
	mGit.On("GetCurrentBranchName", "project-path").Return("some-other-branch", nil)
	mGit.On("Checkout", &u, &p, "project-path", "branch", true).Return(errors.New("FATAL:FAILED"))

	err := CheckoutBranch(&repo, "project-path", "branch", mGit, c, fakeCl)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "FATAL:FAILED") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestCheckoutBranch_ShouldPassForCloneStrategy(t *testing.T) {
	var (
		repo = "repo"
		u    = "user"
		p    = "pass"
	)
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer: "git",
			Repository: &v1alpha1.Repository{
				Url: "repo",
			},
			Strategy: v1alpha1.Import,
		},
	}
	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s, ssh)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c, gs, ssh).Build()

	mGit := new(mockgit.MockGit)
	mGit.On("CheckPermissions", "repo", &u, &p).Return(true)
	mGit.On("GetCurrentBranchName", "project-path").Return("some-other-branch", nil)
	mGit.On("CheckoutRemoteBranchBySSH", "fake", fakeName, "project-path", "branch").Return(nil)

	err := CheckoutBranch(&repo, "project-path", "branch", mGit, c, fakeCl)
	assert.NoError(t, err)
}
