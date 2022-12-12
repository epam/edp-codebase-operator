package handler

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type ImageStreamTagHandler interface {
	ServeRequest(jira *codebaseApi.ImageStreamTag) error
}
