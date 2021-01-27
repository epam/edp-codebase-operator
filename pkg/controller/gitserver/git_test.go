package gitserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBasicAuthEmptyUser(t *testing.T) {
	assert.Nil(t, basicAuth("", ""))
}

func TestBasicAuthFilledUser(t *testing.T) {
	ba := basicAuth("some", "some")
	assert.NotNil(t, ba)
	assert.Equal(t, "some", ba.Username)
	assert.Equal(t, "some", ba.Password)
}
