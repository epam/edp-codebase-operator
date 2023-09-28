package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestCodebaseBranchServiceProvider_AppendVersionToTheHistorySlice(t *testing.T) {
	version := "0-0-1-SNAPSHOT"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			Version: &version,
		},
		Status: codebaseApi.CodebaseBranchStatus{
			VersionHistory: []string{"0-0-0-SNAPSHOT"},
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.AppendVersionToTheHistorySlice(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Equal(t, len(cbResp.Status.VersionHistory), 2)
	assert.Equal(t, cbResp.Status.VersionHistory[1], version)
}

func TestCodebaseBranchServiceProvider_ResetBranchBuildCounter(t *testing.T) {
	b := "100"
	zb := "0"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Build: &b,
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.ResetBranchBuildCounter(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Build, &zb)
}

func TestCodebaseBranchServiceProvider_ResetBranchSuccessBuildCounter(t *testing.T) {
	b := "100"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			LastSuccessfulBuild: &b,
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.ResetBranchSuccessBuildCounter(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Nil(t, cbResp.Status.LastSuccessfulBuild)
}
