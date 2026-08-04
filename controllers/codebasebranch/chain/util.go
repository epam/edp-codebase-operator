package chain

import (
	"fmt"
	"slices"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// HasNewVersion checks if codebase branch has new version.
func HasNewVersion(codebaseBranch *codebaseApi.CodebaseBranch) (bool, error) {
	if codebaseBranch.Spec.Version == nil {
		return false, fmt.Errorf("codebase branch %v doesn't have version", codebaseBranch.Name)
	}

	return !slices.Contains(codebaseBranch.Status.VersionHistory, *codebaseBranch.Spec.Version), nil
}
