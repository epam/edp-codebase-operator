package clean_tmp_directory

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestCleanTempDirectory_ShouldRemoveWithSuccessStatus(t *testing.T) {
	t.Setenv("WORKING_DIR", "/tmp/1")

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "stub-codebase-name",
			BranchName:   "stub-branch-name",
		},
	}
	directory := &CleanTempDirectory{}

	err := directory.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), cb)
	assert.NoError(t, err)
}

func TestCleanTempDirectory_setFailedFields_ShouldPass(t *testing.T) {
	cb := &codebaseApi.CodebaseBranch{}
	setFailedFields(cb, codebaseApi.AcceptCodebaseBranchRegistration, "test")
	assert.Equal(t, cb.Status.DetailedMessage, "test")
}
