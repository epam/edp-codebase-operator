package handler

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type CodebaseImageStreamHandler interface {
	ServeRequest(imageStream *v1alpha1.CodebaseImageStream) error
}
