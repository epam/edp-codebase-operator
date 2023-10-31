package chain

import (
	"fmt"

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
func DirectoryExistsNotEmpty(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}
