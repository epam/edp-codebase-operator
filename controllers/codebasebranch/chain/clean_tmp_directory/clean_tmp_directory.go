package clean_tmp_directory

import (
	"fmt"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type CleanTempDirectory struct {
}

var log = ctrl.Log.WithName("clean-temp-directory-chain")

func (*CleanTempDirectory) ServeRequest(cb *codebaseApi.CodebaseBranch) error {
	rl := log.WithValues("namespace", cb.Namespace, "codebase branch", cb.Name)
	rl.Info("start CleanTempDirectory method...")

	wd := util.GetWorkDir(cb.Spec.CodebaseName, fmt.Sprintf("%v-%v", cb.Namespace, cb.Spec.BranchName))

	if err := deleteWorkDirectory(wd); err != nil {
		setFailedFields(cb, codebaseApi.CleanData, err.Error())

		return err
	}

	rl.Info("end CleanTempDirectory method...")

	return nil
}

func deleteWorkDirectory(dir string) error {
	if err := util.RemoveDirectory(dir); err != nil {
		return errors.Wrapf(err, "couldn't delete directory %v", dir)
	}

	log.Info("directory was cleaned", "path", dir)

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
	}
}
