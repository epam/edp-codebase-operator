package chain

import (
	"fmt"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// HasNewVersion checks if codebase branch has new version.
func HasNewVersion(codebaseBranch *codebaseApi.CodebaseBranch) (bool, error) {
	if codebaseBranch.Spec.Version == nil {
		return false, fmt.Errorf("codebase branch %v doesn't have version", codebaseBranch.Name)
	}

	return !util.SearchVersion(codebaseBranch.Status.VersionHistory, *codebaseBranch.Spec.Version), nil
}
