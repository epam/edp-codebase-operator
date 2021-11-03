package chain

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getIcon(t *testing.T) {
	os.Setenv("ASSETS_DIR", "../../../../build")
	_, err := getIcon()
	assert.Nil(t, err)
}
