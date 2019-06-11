package impl

import (
	"codebase-operator/pkg/apis/edp/v1alpha1"
	"errors"
	"fmt"
	"log"
	"strings"
)

func getRepoUrl(baseUrl string, spec v1alpha1.CodebaseSpec) (*string, error) {
	if spec.Strategy == v1alpha1.Clone {
		log.Printf("Strategy is clone. Try to use default value...")
		return tryGetRepoUrl(spec)
	}
	log.Printf("Strategy is not clone. Start build url...")
	url := buildRepoUrl(baseUrl, spec)
	log.Printf("Url has been generated: %v", url)
	return &url, nil

}

func tryGetRepoUrl(spec v1alpha1.CodebaseSpec) (*string, error) {
	if spec.Repository == nil {
		return nil, errors.New("repository cannot be nil for specified strategy")
	}
	return &spec.Repository.Url, nil
}

func buildRepoUrl(baseUrl string, spec v1alpha1.CodebaseSpec) string {
	log.Printf("Start build repo url by base url: %v and spec %+v", baseUrl, spec)
	var result string
	if spec.Type == "application" {
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
	frameworks := []string{"springboot", "react", "netcore"}
	if lang == "java" {
		return frameworks[0]
	} else if lang == "javascript" {
		return frameworks[1]
	}
	return frameworks[2]
}
