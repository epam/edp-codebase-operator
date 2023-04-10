package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func Test_getIcon(t *testing.T) {
	t.Setenv(util.AssetsDirEnv, "../../../build")

	_, err := getIcon()
	assert.NoError(t, err)
}
