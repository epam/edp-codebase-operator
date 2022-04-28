package model

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestConvertToGitServer(t *testing.T) {
	gs := ConvertToGitServer(v1alpha1.GitServer{
		Status: v1alpha1.GitServerStatus{
			Status: "hello world",
		},
	})

	assert.Equal(t, gs.ActionLog.Event, "hello_world")
}
