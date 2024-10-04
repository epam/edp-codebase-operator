package gitprovider

import (
	"context"
	"fmt"
	"net/http"
)

type BearerTokenAuthProvider struct {
	token string
}

func NewBearerTokenAuthProvider(token string) *BearerTokenAuthProvider {
	return &BearerTokenAuthProvider{token: token}
}

func (s *BearerTokenAuthProvider) Intercept(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	return nil
}

type BasicTokenAuthProvider struct {
	token string
}

func NewBasicTokenAuthProvider(token string) *BasicTokenAuthProvider {
	return &BasicTokenAuthProvider{token: token}
}

func (s *BasicTokenAuthProvider) Intercept(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", s.token))

	return nil
}
