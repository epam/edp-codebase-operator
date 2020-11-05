package clean_tmp_directory

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCleanTempDirectory_ShouldRemoveWithSuccessStatus(t *testing.T) {
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "stub-codebase-name",
			BranchName:   "stub-branch-name",
		},
	}
	directory := CleanTempDirectory{}
	err := directory.ServeRequest(cb)
	assert.NoError(t, err)
}

func TestCleanTempDirectory_ShouldThrowError(t *testing.T) {
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "stub-name",
			BranchName:   ".",
		},
	}
	directory := CleanTempDirectory{}
	err := directory.ServeRequest(cb)
	assert.Error(t, err)
	assert.Equal(t, v1alpha1.CleanData, cb.Status.Action)
}
