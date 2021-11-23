package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionFromBuild(t *testing.T) {
	// prepare
	version = "0.0.2" // set during the build
	defer func() {
		version = ""
	}()

	assert.Contains(t, Get().String(), "0.0.2")
}
