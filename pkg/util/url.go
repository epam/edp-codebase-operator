package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

var lf = map[string]string{
	"javascript":      "react",
	"groovy-pipeline": "codenarc",
	"dotnet":          "netcore",
	"python":          "python-3.8",
	"terraform":       "terraform",
	"rego":            "opa",
}

func GetRepoUrl(c *v1alpha1.Codebase) (*string, error) {
	log.Info("Setup repo url", "codebase_name", c.Name)
	if c.Spec.Strategy == v1alpha1.Clone {
		log.Info("strategy is clone. Try to use default value...", "codebase_name", c.Name)
		return tryGetRepoUrl(c.Spec)
	}

	log.Info("Strategy is not clone. Start build url...", "codebase_name", c.Name)
	u := BuildRepoUrl(c.Spec)
	log.Info("ApiUrl has been generated", "url", u, "codebase_name", c.Name)
	return &u, nil

}

func tryGetRepoUrl(spec v1alpha1.CodebaseSpec) (*string, error) {
	if spec.Repository == nil {
		return nil, errors.New("repository cannot be nil for specified strategy")
	}
	return &spec.Repository.Url, nil
}

func BuildRepoUrl(spec v1alpha1.CodebaseSpec) string {
	log.Info("Start building repo url", "base url", GithubDomain, "spec", spec)
	return strings.ToLower(fmt.Sprintf("%v/%v-%v-%v.git", GithubDomain, spec.Lang, spec.BuildTool,
		getFrameworkOrDefault(spec)))
}

func getFrameworkOrDefault(spec v1alpha1.CodebaseSpec) string {
	if spec.Framework != nil && *spec.Framework != "" {
		return *spec.Framework
	}
	return lf[strings.ToLower(spec.Lang)]
}
