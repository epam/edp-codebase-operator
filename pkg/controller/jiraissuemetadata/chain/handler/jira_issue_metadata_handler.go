package handler

import edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"

type JiraIssueMetadataHandler interface {
	ServeRequest(version *edpv1alpha1.JiraIssueMetadata) error
}
