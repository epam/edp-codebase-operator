package chain

import (
	"fmt"
	"path"

	"golang.org/x/exp/slices"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// HasNewVersion checks if codebase branch has new version.
func HasNewVersion(codebaseBranch *codebaseApi.CodebaseBranch) (bool, error) {
	if codebaseBranch.Spec.Version == nil {
		return false, fmt.Errorf("codebase branch %v doesn't have version", codebaseBranch.Name)
	}

	return !slices.Contains(codebaseBranch.Status.VersionHistory, *codebaseBranch.Spec.Version), nil
}

// DirectoryExistsNotEmpty checks if directory exists and not empty.
func DirectoryExistsNotEmpty(dirPath string) bool {
	return util.DoesDirectoryExist(dirPath) && !util.IsDirectoryEmpty(dirPath)
}

func GetCodebaseBranchWorkingDirectory(codebaseBranch *codebaseApi.CodebaseBranch) string {
	return path.Join(
		util.GetWorkDir(codebaseBranch.Spec.CodebaseName, codebaseBranch.Namespace),
		"codebase-branches",
		codebaseBranch.Name,
	)
}
