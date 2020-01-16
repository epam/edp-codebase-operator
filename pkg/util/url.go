package util

import (
	"errors"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"strings"
)

func GetRepoUrl(baseUrl string, c *v1alpha1.Codebase) (*string, error) {
	log.Info("Setup repo url", "codebase name", c.Name)
	if c.Spec.Strategy == v1alpha1.Clone {
		log.Info("strategy is clone. Try to use default value...", "codebase name", c.Name)
		return tryGetRepoUrl(c.Spec)
	}

	log.Info("Strategy is not clone. Start build url...", "codebase name", c.Name)
	url := buildRepoUrl(baseUrl, c.Spec)
	log.Info("Url has been generated", "url", url, "codebase name", c.Name)
	return &url, nil

}

func tryGetRepoUrl(spec v1alpha1.CodebaseSpec) (*string, error) {
	if spec.Repository == nil {
		return nil, errors.New("repository cannot be nil for specified strategy")
	}
	return &spec.Repository.Url, nil
}

func buildRepoUrl(baseUrl string, spec v1alpha1.CodebaseSpec) string {
	log.Info("Start building repo url", "base url", baseUrl, "spec", spec)
	var result string
	if spec.Type == Application {
		result = fmt.Sprintf("%v/%v-%v-%v",
			baseUrl, spec.Lang, spec.BuildTool, *spec.Framework)
	} else if spec.Type == "autotests" {
		result = fmt.Sprintf("%v/%v-%v",
			baseUrl, spec.Lang, spec.BuildTool)
	} else if spec.Type == "library" {
		result = fmt.Sprintf("%v/%v-%v-%v",
			baseUrl, spec.Lang, spec.BuildTool, setLibraryFramework(strings.ToLower(spec.Lang)))
	}

	if spec.Database != nil {
		result += "-" + spec.Database.Kind
	}

	return strings.ToLower(result + ".git")
}

func setLibraryFramework(lang string) string {
	frameworks := []string{"springboot", "react", "groovy", "netcore"}
	if lang == "java" {
		return frameworks[0]
	} else if lang == "javascript" {
		return frameworks[1]
	} else if lang == "groovy-pipeline" {
		return frameworks[2]
	}
	return frameworks[3]
}
