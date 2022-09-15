package chain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	mockGit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestCloneGitProject_ShouldPass(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitUrlPath: util.GetStringP(fakeName),
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			GitServer: fakeName,
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

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s, ssh)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c, gs, ssh).Build()
	mGit := new(mockGit.MockGit)
	var port int32 = 22
	wd := util.GetWorkDir(fakeName, fakeNamespace)
	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("ssh://%v:22%v", fakeName, fakeName),
		wd, port).Return(
		nil)

	cgp := NewCloneGitProject(
		fakeCl,
		mGit,
	)

	err = cgp.ServeRequest(ctx, c)
	assert.NoError(t, err)
}

func TestCloneGitProject_SetIntermediateSuccessFieldsShouldFail(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name: fakeName,
		},
	}
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	cgp := NewCloneGitProject(
		fakeCl,
		nil,
	)

	err := cgp.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "an error has been occurred while updating fake-name Codebase status", "wrong error returned")
}

func TestCloneGitProject_GetGitServerShouldFail(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitUrlPath: util.GetStringP(fakeName),
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			GitServer: fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	cgp := NewCloneGitProject(
		fakeCl,
		nil,
	)

	err := cgp.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "an error has occurred while getting fake-name GitServer", "wrong error returned")
}

func TestCloneGitProject_GetSecretShouldFail(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitUrlPath: util.GetStringP(fakeName),
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			GitServer: fakeName,
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs).Build()

	cgp := NewCloneGitProject(
		fakeCl,
		nil,
	)

	err := cgp.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "an error has occurred while getting fake-name secret", "wrong error returned")
}

func TestCloneGitProject_CloneRepositoryBySshShouldFail(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitUrlPath: util.GetStringP(fakeName),
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			GitServer: fakeName,
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

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s, ssh)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s, c, gs, ssh).Build()
	mGit := new(mockGit.MockGit)
	var port int32 = 22
	wd := util.GetWorkDir(fakeName, fakeNamespace)
	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("ssh://%v:22%v", fakeName, fakeName),
		wd, port).Return(
		errors.New("FATAL ERROR"))

	cgp := NewCloneGitProject(
		fakeCl,
		mGit,
	)

	err = cgp.ServeRequest(ctx, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "an error has occurred while cloning repository ssh://fake-name:22fake-name: FATAL ERROR", "wrong error returned")
}

func TestCloneGitProject_Postpone(t *testing.T) {
	ctx := context.Background()
	cl := NewCloneGitProject(nil, nil)
	p := repoNotReady

	err := cl.ServeRequest(ctx, &codebaseApi.Codebase{Spec: codebaseApi.CodebaseSpec{GitUrlPath: &p}})
	assert.Error(t, err)
	assert.IsType(t, PostponeError{}, err)
}
