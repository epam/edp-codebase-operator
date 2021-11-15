package put_branch_in_git

import (
	"fmt"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-perf-operator/v2/pkg/util/common"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	fakeName       = "fake-name"
	fakeNamespace  = "fake-namespace"
	versioningType = "edp"
)

func TestPutBranchInGit_ShouldBeExecutedSuccessfullyWithDefaultVersioning(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: common.GetStringP(fakeName),
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
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
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs, cb)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, cb, s).Build()

	mGit := new(mock.MockGit)
	var port int32 = 22
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))
	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("%v:%v", fakeName, fakeName),
		wd, port).Return(
		nil)

	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName).Return(
		nil)

	err := PutBranchInGit{
		Client: fakeCl,
		Git:    mGit,
	}.ServeRequest(cb)

	assert.NoError(t, err)
}

func TestPutBranchInGit_CodebaseShouldNotBeFound(t *testing.T) {
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
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
	assert.Equal(t, v1alpha1.PutBranchForGitlabCiCodebase, cb.Status.Action)
}

func TestPutBranchInGit_ShouldThrowCodebaseBranchReconcileError(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: common.GetStringP(fakeName),
		},
		Status: v1alpha1.CodebaseStatus{
			Available: false,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	err := PutBranchInGit{
		Client: fakeCl,
	}.ServeRequest(cb)

	assert.ErrorIs(t, err, err.(*util.CodebaseBranchReconcileError))
}

func TestPutBranchInGit_ShouldBeExecutedSuccessfullyWithEdpVersioning(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: common.GetStringP(fakeName),
			Versioning: v1alpha1.Versioning{
				Type:      versioningType,
				StartFrom: nil,
			},
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
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
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"keyName": []byte("fake"),
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			Version:      common.GetStringP("version3"),
			BranchName:   fakeName,
		},
		Status: v1alpha1.CodebaseBranchStatus{
			VersionHistory: []string{"version1", "version2"},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs, cb)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, cb, s).Build()

	mGit := new(mock.MockGit)
	var port int32 = 22
	wd := util.GetWorkDir(fakeName, fmt.Sprintf("%v-%v", fakeNamespace, fakeName))
	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("%v:%v", fakeName, fakeName),
		wd, port).Return(
		nil)

	mGit.On("CreateRemoteBranch", "", fakeName, wd, fakeName).Return(
		nil)

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

	cb := &v1alpha1.CodebaseBranch{}

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
	c := &v1alpha1.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: common.GetStringP(fakeName),
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
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
	c := &v1alpha1.Codebase{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer:  fakeName,
			GitUrlPath: common.GetStringP(fakeName),
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
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
