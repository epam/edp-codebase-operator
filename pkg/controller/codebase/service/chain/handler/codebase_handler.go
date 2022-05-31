package handler

import codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"

type CodebaseHandler interface {
	ServeRequest(c *codebaseApi.Codebase) error
}
