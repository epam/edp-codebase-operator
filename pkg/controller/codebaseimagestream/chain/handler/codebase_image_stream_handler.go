package handler

import (
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

type CodebaseImageStreamHandler interface {
	ServeRequest(imageStream *codebaseApi.CodebaseImageStream) error
}
