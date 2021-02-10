package chain

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
