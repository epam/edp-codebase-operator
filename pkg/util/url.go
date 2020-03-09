package util

import (
	"errors"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"strings"
)

var lf = map[string]string{
	"javascript":      "react",
	"groovy-pipeline": "groovy",
	"dotnet":          "netcore",
}

func GetRepoUrl(c *v1alpha1.Codebase) (*string, error) {
	log.Info("Setup repo url", "codebase name", c.Name)
	if c.Spec.Strategy == v1alpha1.Clone {
		log.Info("strategy is clone. Try to use default value...", "codebase name", c.Name)
		return tryGetRepoUrl(c.Spec)
	}

	log.Info("Strategy is not clone. Start build url...", "codebase name", c.Name)
	url := buildRepoUrl(c.Spec)
	log.Info("Url has been generated", "url", url, "codebase name", c.Name)
	return &url, nil

}

func tryGetRepoUrl(spec v1alpha1.CodebaseSpec) (*string, error) {
	if spec.Repository == nil {
		return nil, errors.New("repository cannot be nil for specified strategy")
	}
	return &spec.Repository.Url, nil
}

func buildRepoUrl(spec v1alpha1.CodebaseSpec) string {
	log.Info("Start building repo url", "base url", GithubDomain, "spec", spec)
	return strings.ToLower(fmt.Sprintf("%v/%v-%v-%v%v.git", GithubDomain, spec.Lang, spec.BuildTool,
		getFrameworkOrEmpty(spec), getDatabaseOrEmpty(spec.Database)))
}

func getDatabaseOrEmpty(db *v1alpha1.Database) string {
	if db != nil {
		return fmt.Sprintf("-%v", db.Kind)
	}
	return ""
}

func getFrameworkOrEmpty(spec v1alpha1.CodebaseSpec) string {
	if spec.Framework != nil && *spec.Framework != "" {
		return *spec.Framework
	}
	return lf[strings.ToLower(spec.Lang)]
}
