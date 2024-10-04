package gitprovider

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicTokenAuthProvider_Intercept(t *testing.T) {
	p := NewBasicTokenAuthProvider("test")
	req, _ := http.NewRequest("GET", "https://test.com", http.NoBody)
	err := p.Intercept(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Basic test", req.Header.Get("Authorization"))
}

func TestBearerTokenAuthProvider_Intercept(t *testing.T) {
	p := NewBearerTokenAuthProvider("test")
	req, _ := http.NewRequest("GET", "https://test.com", http.NoBody)
	err := p.Intercept(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer test", req.Header.Get("Authorization"))
}
