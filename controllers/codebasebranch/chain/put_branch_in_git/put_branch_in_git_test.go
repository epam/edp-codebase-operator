package put_branch_in_git

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	gitServerMocks "github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	fakeName       = "fake-name"
	fakeNamespace  = "fake-namespace"
	versioningType = "edp"
)

func TestPutBranchInGit_ShouldBeExecutedSuccessfullyWithDefaultVersioning(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: v1.ObjectMeta{
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

	s := &coreV1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
			FromCommit:   "commitsha",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs, cb)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, cb, s).Build()

	mGit := gitServerMocks.NewGit(t)
	port := int32(22)
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))

	repoSshUrl := util.GetSSHUrl(gs, c.Spec.GetProjectID())

	mGit.On("CloneRepositoryBySsh", testifymock.Anything, "", fakeName, repoSshUrl, wd, port).Return(nil)
	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName, "commitsha", port).Return(nil)

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    mGit,
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	assert.NoError(t, err)
}

func TestPutBranchInGit_CodebaseShouldNotBeFound(t *testing.T) {
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch Codebase")
	assert.Equal(t, codebaseApi.PutGitBranch, cb.Status.Action)
}

func TestPutBranchInGit_ShouldThrowCodebaseBranchReconcileError(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Available: false,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	_, ok := err.(*util.CodebaseBranchReconcileError)
	assert.True(t, ok, "wrong type of error")
}

func TestPutBranchInGit_ShouldBeExecutedSuccessfullyWithEdpVersioning(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: fakeName,
			Versioning: codebaseApi.Versioning{
				Type:      versioningType,
				StartFrom: nil,
			},
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: v1.ObjectMeta{
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

	s := &coreV1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			Version:      util.GetStringP("version3"),
			BranchName:   fakeName,
			FromCommit:   "",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			VersionHistory: []string{"version1", "version2"},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs, cb)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, cb, s).Build()

	mGit := gitServerMocks.NewGit(t)

	port := int32(22)
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))

	repoSshUrl := util.GetSSHUrl(gs, c.Spec.GetProjectID())

	mGit.On("CloneRepositoryBySsh", testifymock.Anything, "", fakeName, repoSshUrl, wd, port).
		Return(nil)
	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName, "", port).Return(nil)

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    mGit,
		Service: &service.CodebaseBranchServiceProvider{
			Client: fakeCl,
		},
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	assert.NoError(t, err)
}

func TestPutBranchInGit_ShouldFailToSetIntermediateStatus(t *testing.T) {
	cb := &codebaseApi.CodebaseBranch{}

	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	err := PutBranchInGit{
		Client: fakeCl,
		Service: &service.CodebaseBranchServiceProvider{
			Client: fakeCl,
		},
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	assert.Error(t, err)
}

func TestPutBranchInGit_GitServerShouldNotBeFound(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, coreV1.AddToScheme(scheme))
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "failed to fetch GitServer")
}

func TestPutBranchInGit_SecretShouldNotBeFound(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: v1.ObjectMeta{
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

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs, cb)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, &coreV1.Secret{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("failed to get %s secret", fakeName))
}

func TestPutBranchInGit_ShouldFailNoEDPVersion(t *testing.T) {
	codeBase := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: fakeName,
			Versioning: codebaseApi.Versioning{
				Type: "edp",
			},
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	gitServer := &codebaseApi.GitServer{
		ObjectMeta: v1.ObjectMeta{
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

	secret := &coreV1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	codeBaseBranch := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, codeBase, gitServer, codeBaseBranch)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, secret)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(codeBase, gitServer, codeBaseBranch, secret).Build()

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    gitServerMocks.NewGit(t),
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), codeBaseBranch)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't have version")
}

func TestPutBranchInGit_SkipAlreadyCreated(t *testing.T) {
	codeBaseBranch := &codebaseApi.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fake",
			Namespace: "default",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "fake",
			BranchName:   "fake",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Git: codebaseApi.CodebaseBranchGitStatusBranchCreated,
		},
	}

	err := PutBranchInGit{
		Client: fake.NewClientBuilder().Build(),
		Git:    gitServerMocks.NewGit(t),
	}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), codeBaseBranch)

	require.NoError(t, err)
}
