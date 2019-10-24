package model

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type GerritConfigGoTemplating struct {
	Lang              string             `json:"lang"`
	Framework         *string            `json:"framework"`
	BuildTool         string             `json:"build_tool"`
	RepositoryUrl     *string            `json:"repository_url"`
	Route             *v1alpha1.Route    `json:"route"`
	Database          *v1alpha1.Database `json:"database"`
	CodebaseSettings  CodebaseSettings   `json:"app_settings"`
	DockerRegistryUrl string             `json:"docker_registry_url"`
	TemplatesDir      string             `json:"templates_dir"`
	CloneSshUrl       string             `json:"clone_ssh_url"`
}
