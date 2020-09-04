package handler

import "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"

type GitTagHandler interface {
	ServeRequest(jira *v1alpha1.GitTag) error
}
