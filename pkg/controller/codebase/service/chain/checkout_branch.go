package chain

import (
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/pkg/errors"
)

func CheckoutBranch(projectPath, branchName string, git git.Git) error {
	currentBranchName, err := git.GetCurrentBranchName(projectPath)
	if err != nil {
		return err
	}
	if currentBranchName == branchName {
		log.Info("default branch is already active", "name", branchName)
		return nil
	}
	if err := git.Checkout(projectPath, branchName); err != nil {
		return errors.Wrapf(err, "checkout default branch %v has been failed", branchName)
	}
	return nil
}