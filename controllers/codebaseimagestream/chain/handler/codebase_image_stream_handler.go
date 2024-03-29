package handler

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type CodebaseImageStreamHandler interface {
	ServeRequest(imageStream *codebaseApi.CodebaseImageStream) error
}
