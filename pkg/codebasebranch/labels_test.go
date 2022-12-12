package codebasebranch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestAddCodebaseLabel_OK(t *testing.T) {
	ctx := context.Background()
	cbBranchName_1 := "NewCodebaseBranch"
	namespace := "namespace"
	cbName_1 := "NewCodebase"
	branchName := "master"

	codebaseBranch := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cbBranchName_1,
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: cbName_1,
			BranchName:   branchName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, codebaseBranch)
	fakeK8sClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(codebaseBranch).Build()

	err := AddCodebaseLabel(ctx, fakeK8sClient, codebaseBranch, cbName_1)
	assert.NoError(t, err)

	gotCodebaseBranch := &codebaseApi.CodebaseBranch{}

	err = fakeK8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      cbBranchName_1,
		Namespace: namespace,
	}, gotCodebaseBranch)
	if err != nil {
		t.Fatal(err)
	}

	expectedLabels := map[string]string{
		LabelCodebaseName: cbName_1,
	}
	assert.Equal(t, expectedLabels, gotCodebaseBranch.GetLabels())
}

func TestAddCodebaseLabel_LabelExists(t *testing.T) {
	ctx := context.Background()
	cbBranchName_1 := "NewCodebaseBranch"
	namespace := "namespace"
	cbName_1 := "NewCodebase"
	branchName := "master"

	codebaseBranch := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cbBranchName_1,
			Namespace: namespace,
			Labels:    map[string]string{LabelCodebaseName: cbName_1},
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: cbName_1,
			BranchName:   branchName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, codebaseBranch)
	fakeK8sClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(codebaseBranch).Build()

	err := AddCodebaseLabel(ctx, fakeK8sClient, codebaseBranch, cbName_1)
	assert.NoError(t, err)

	gotCodebaseBranch := &codebaseApi.CodebaseBranch{}

	err = fakeK8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      cbBranchName_1,
		Namespace: namespace,
	}, gotCodebaseBranch)
	if err != nil {
		t.Fatal(err)
	}

	expectedLabels := map[string]string{
		LabelCodebaseName: cbName_1,
	}
	assert.Equal(t, expectedLabels, gotCodebaseBranch.GetLabels())
}
