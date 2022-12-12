package chain

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getIcon(t *testing.T) {
	err := os.Setenv("ASSETS_DIR", "../../../build")
	require.NoError(t, err)

	_, err = getIcon()
	assert.Nil(t, err)
}
