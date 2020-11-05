package put_branch_in_git

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epmd-edp/perf-operator/v2/pkg/util/common"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
	objs := []runtime.Object{
		c, gs, s,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs)

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	mGit := new(mock.MockGit)

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/", fakeNamespace, fakeName)
	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("%v:%v%v", fakeName, 22, fakeName),
		wd).Return(
		nil)

	mGit.On("CreateRemoteBranch", "", fakeName, wd, "").Return(
		nil)

	err := PutBranchInGit{
		Client: fake.NewFakeClient(objs...),
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

	err := PutBranchInGit{
		Client: fake.NewFakeClient([]runtime.Object{}...),
	}.ServeRequest(cb)

	assert.Error(t, err)
	assert.Equal(t, v1alpha1.PutBranchForGitlabCiCodebase, cb.Status.Action)
}

func TestPutBranchInGit_ShouldThrowNCodebaseBranchReconcileError(t *testing.T) {
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

	objs := []runtime.Object{
		c,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c)

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	err := PutBranchInGit{
		Client: fake.NewFakeClient(objs...),
	}.ServeRequest(cb)

	assert.Error(t, err)
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
		},
		Status: v1alpha1.CodebaseBranchStatus{
			VersionHistory: []string{"version1", "version2"},
		},
	}

	objs := []runtime.Object{
		c, gs, s, cb,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs, cb)

	mGit := new(mock.MockGit)

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/", fakeNamespace, fakeName)
	mGit.On("CloneRepositoryBySsh", "",
		fakeName, fmt.Sprintf("%v:%v%v", fakeName, 22, fakeName),
		wd).Return(
		nil)

	mGit.On("CreateRemoteBranch", "", fakeName, wd, "").Return(
		nil)

	client := fake.NewFakeClient(objs...)
	err := PutBranchInGit{
		Client: client,
		Git:    mGit,
		Service: service.CodebaseBranchService{
			Client: client,
		},
	}.ServeRequest(cb)

	assert.NoError(t, err)
}

func TestPutBranchInGit_ShouldThrowErrorWithEdpVersioning(t *testing.T) {
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

	objs := []runtime.Object{
		c,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, &v1alpha1.CodebaseBranch{})

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			Version:      common.GetStringP("version3"),
		},
		Status: v1alpha1.CodebaseBranchStatus{
			VersionHistory: []string{"version1", "version2"},
			Build:          common.GetStringP("0"),
		},
	}

	client := fake.NewFakeClient(objs...)
	err := PutBranchInGit{
		Client: client,
		Service: service.CodebaseBranchService{
			Client: client,
		},
	}.ServeRequest(cb)

	assert.Error(t, err)
}

func TestPutBranchInGit_GitServerShouldNitBeFound(t *testing.T) {
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

	objs := []runtime.Object{
		c,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c)

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	err := PutBranchInGit{
		Client: fake.NewFakeClient(objs...),
	}.ServeRequest(cb)

	assert.Error(t, err)
}

func TestPutBranchInGit_SecretShouldNitBeFound(t *testing.T) {
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

	objs := []runtime.Object{
		c, gs,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, c, gs)

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	err := PutBranchInGit{
		Client: fake.NewFakeClient(objs...),
	}.ServeRequest(cb)

	assert.Error(t, err)
}
