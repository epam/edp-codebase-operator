package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	token, err := GenerateRandomString(20)

	assert.NoError(t, err)

	nextToken, err := GenerateRandomString(20)

	assert.NoError(t, err)
	assert.NotEqual(t, token, nextToken)
}
