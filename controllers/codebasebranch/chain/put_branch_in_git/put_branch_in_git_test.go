package put_branch_in_git

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	gitServerMocks "github.com/epam/edp-codebase-operator/v2/controllers/gitserver/mocks"
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
			GitUrlPath: util.GetStringP(fakeName),
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

	mGit := new(gitServerMocks.MockGit)
	port := int32(22)
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))

	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("%v:%v", fakeName, fakeName),
		wd, port).Return(nil)
	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName, "commitsha", port).Return(nil)

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    mGit,
	}.ServeRequest(cb)

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
	scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(cb)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "Unable to get Codebase fake-name") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Equal(t, codebaseApi.PutBranchForGitlabCiCodebase, cb.Status.Action)
}

func TestPutBranchInGit_ShouldThrowCodebaseBranchReconcileError(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: util.GetStringP(fakeName),
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
	}.ServeRequest(cb)

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
			GitUrlPath: util.GetStringP(fakeName),
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

	mGit := new(gitServerMocks.MockGit)

	port := int32(22)
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))

	mGit.On("CloneRepositoryBySsh", "", fakeName, fmt.Sprintf("%v:%v", fakeName, fakeName), wd, port).
		Return(nil)
	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName, "", port).Return(nil)

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    mGit,
		Service: &service.CodebaseBranchServiceProvider{
			Client: fakeCl,
		},
	}.ServeRequest(cb)

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
	}.ServeRequest(cb)

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
			GitUrlPath: util.GetStringP(fakeName),
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
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(cb)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "an error has occurred while getting fake-name Git Server CR") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutBranchInGit_SecretShouldNotBeFound(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: util.GetStringP(fakeName),
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
	}.ServeRequest(cb)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "Unable to get secret fake-name") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutBranchInGit_ShouldFailNoEDPVersion(t *testing.T) {
	codeBase := &codebaseApi.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: util.GetStringP(fakeName),
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

	mGit := &gitServerMocks.MockGit{}
	port := int32(22)
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))

	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("%v:%v", fakeName, fakeName),
		wd, port).Return(nil)
	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName, "commitsha", port).Return(nil)

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    mGit,
	}.ServeRequest(codeBaseBranch)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't have version")
}
