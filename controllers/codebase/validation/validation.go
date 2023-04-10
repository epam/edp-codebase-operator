package validation

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

func IsCodebaseValid(cr *codebaseApi.Codebase) error {
	if !containSettings(allowedCodebaseSettings["add_repo_strategy"], string(cr.Spec.Strategy)) {
		return fmt.Errorf("provided unsupported repository strategy: %s", string(cr.Spec.Strategy))
	}

	if !containSettings(allowedCodebaseSettings["language"], cr.Spec.Lang) {
		return fmt.Errorf("provided unsupported language: %s", cr.Spec.Lang)
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
