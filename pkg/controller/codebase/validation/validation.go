package validation

import (
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

var log = ctrl.Log.WithName("codebase_validator")

var allowedCodebaseSettings = map[string][]string{
	"add_repo_strategy": {"create", "clone", "import"},
	"language":          {"java", "dotnet", "javascript", "groovy-pipeline", "other", "go", "python", "terraform", "rego"},
}

func IsCodebaseValid(cr *edpv1alpha1.Codebase) bool {
	if !(containSettings(allowedCodebaseSettings["add_repo_strategy"], string(cr.Spec.Strategy))) {
		log.Info("Provided unsupported repository strategy", "strategy", string(cr.Spec.Strategy))
		return false
	} else if !(containSettings(allowedCodebaseSettings["language"], cr.Spec.Lang)) {
		log.Info("Provided unsupported language", "language", cr.Spec.Lang)
		return false
	}
	return true
}

func containSettings(slice []string, value string) bool {
	for _, element := range slice {
		if element == strings.ToLower(value) {
			return true
		}
	}
	return false
}
