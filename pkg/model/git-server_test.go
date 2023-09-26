package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestConvertToGitServer(t *testing.T) {
	t.Parallel()

	gs := ConvertToGitServer(&codebaseApi.GitServer{
		Spec: codebaseApi.GitServerSpec{
			GitHost: "github.com",
		},
	})

	assert.Equal(t, gs.GitHost, "github.com")
}
