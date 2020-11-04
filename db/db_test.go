package db

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetConnectionDBEnabledFalse(t *testing.T) {
	_ = os.Setenv("DB_ENABLED", "false")
	db := GetConnection()

	assert.Nil(t, db)
}

func TestGetConnectionDBEnabledTrue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Should be panic because of absent DB_HOST env variable")
		}
	}()

	_ = os.Setenv("DB_ENABLED", "true")
	_ = GetConnection()
}
