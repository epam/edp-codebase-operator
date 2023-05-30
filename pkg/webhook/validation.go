package webhook

import (
	"fmt"
	"strings"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

var allowedCodebaseSettings = map[string][]string{
	"add_repo_strategy": {"create", "clone", "import"},
	"language": {"java", "dotnet", "javascript",
		"groovy-pipeline", "other", "go", "python", "terraform", "rego", "container", "helm", "csharp"},
}

func IsCodebaseValid(codebase *codebaseApi.Codebase) error {
	if !containSettings(allowedCodebaseSettings["add_repo_strategy"], string(codebase.Spec.Strategy)) {
		return fmt.Errorf("provided unsupported repository strategy: %s", string(codebase.Spec.Strategy))
	}

	if !containSettings(allowedCodebaseSettings["language"], codebase.Spec.Lang) {
		return fmt.Errorf("provided unsupported language: %s", codebase.Spec.Lang)
	}

	if codebase.Spec.Versioning.Type != codebaseApi.VersioningTypDefault &&
		(codebase.Spec.Versioning.StartFrom == nil || *codebase.Spec.Versioning.StartFrom == "") {
		return fmt.Errorf("versioning start from is required when versioning type is not default")
	}

	return nil
}

func containSettings(slice []string, value string) bool {
	for _, element := range slice {
		if strings.EqualFold(element, value) {
			return true
		}
	}

	return false
}
