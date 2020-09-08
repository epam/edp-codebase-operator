package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
)

type CleanTempDirectory struct {
}

func (h CleanTempDirectory) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	rl := log.WithValues("namespace", cb.Namespace, "codebase branch", cb.Name)
	rl.Info("start CleanTempDirectory method...")

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v", cb.Namespace, cb.Spec.CodebaseName, cb.Spec.BranchName)
	if err := deleteWorkDirectory(wd); err != nil {
		setFailedFields(cb, v1alpha1.CleanData, err.Error())
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
