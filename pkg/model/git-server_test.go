package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

func TestConvertToGitServer(t *testing.T) {
	gs := ConvertToGitServer(codebaseApi.GitServer{
		Status: codebaseApi.GitServerStatus{
			Status: "hello world",
		},
	})

	assert.Equal(t, gs.ActionLog.Event, "hello_world")
}
