package clean_tmp_directory

import (
	"context"
	"fmt"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type CleanTempDirectory struct{}

func (*CleanTempDirectory) ServeRequest(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx).WithName("clean-temp-directory")

	log.Info("Start CleanTempDirectory method")

	wd := chain.GetCodebaseBranchWorkingDirectory(cb)

	if err := deleteWorkDirectory(wd); err != nil {
		setFailedFields(cb, codebaseApi.CleanData, err.Error())

		return err
	}

	log.Info("End cleaning temp directory")

	return nil
}

func deleteWorkDirectory(dir string) error {
	if err := util.RemoveDirectory(dir); err != nil {
		return fmt.Errorf("failed to delete directory %v: %w", dir, err)
	}

	return nil
}

func setFailedFields(cb *codebaseApi.CodebaseBranch, a codebaseApi.ActionType, message string) {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:          util.StatusFailed,
		LastTimeUpdated: metaV1.Now(),
		Username:        "system",
		Action:          a,
		Result:          codebaseApi.Error,
		DetailedMessage: message,
		Value:           "failed",
		Git:             cb.Status.Git,
		VersionHistory:  cb.Status.VersionHistory,
		Build:           cb.Status.Build,
	}
}
