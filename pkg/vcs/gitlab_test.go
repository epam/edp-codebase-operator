package vcs

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitLabClient_CreateWebHook(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name       string
		respStatus int
		resBody    map[string]interface{}
		want       *WebHook
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name:       "success",
			respStatus: http.StatusCreated,
			resBody:    map[string]interface{}{"id": 1, "url": "https://example.com"},
			want:       &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:    assert.NoError,
		},
		{
			name:       "failure",
			respStatus: http.StatusBadRequest,
			resBody:    map[string]interface{}{"message": "bad request"},
			wantErr:    assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder, err := httpmock.NewJsonResponder(tt.respStatus, tt.resBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, responder)

			c := NewGitLabClient(restyClient)

			got, err := c.CreateWebHook(context.Background(), "url", "token", "project", "secret", "webHookURL")
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitLabClient_GetWebHook(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	fakeUrlRegexp := regexp.MustCompile(`.*`)

	tests := []struct {
		name       string
		respStatus int
		resBody    map[string]interface{}
		want       *WebHook
		wantErr    assert.ErrorAssertionFunc
		errIs      error
	}{
		{
			name:       "success",
			respStatus: http.StatusOK,
			resBody:    map[string]interface{}{"id": 1, "url": "https://example.com"},
			want:       &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:    assert.NoError,
		},
		{
			name:       "not found",
			respStatus: http.StatusNotFound,
			resBody:    map[string]interface{}{"message": "not found"},
			wantErr:    assert.Error,
			errIs:      ErrWebHookNotFound,
		},
		{
			name:       "failure",
			respStatus: http.StatusBadRequest,
			resBody:    map[string]interface{}{"message": "bad request"},
			wantErr:    assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder, err := httpmock.NewJsonResponder(tt.respStatus, tt.resBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, responder)

			c := NewGitLabClient(restyClient)

			got, err := c.GetWebHook(context.Background(), "url", "token", "project", 999)
			if !tt.wantErr(t, err) {
				return
			}
			if tt.errIs != nil {
				assert.ErrorIs(t, err, tt.errIs)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitLabClient_DeleteWebHook(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name       string
		respStatus int
		wantErr    assert.ErrorAssertionFunc
		errIs      error
	}{
		{
			name:       "success",
			respStatus: http.StatusOK,
			wantErr:    assert.NoError,
		},
		{
			name:       "not found",
			respStatus: http.StatusNotFound,
			wantErr:    assert.Error,
			errIs:      ErrWebHookNotFound,
		},
		{
			name:       "failure",
			respStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder := httpmock.NewStringResponder(tt.respStatus, "")
			httpmock.RegisterRegexpResponder(http.MethodDelete, fakeUrlRegexp, responder)

			c := NewGitLabClient(restyClient)

			err := c.DeleteWebHook(context.Background(), "url", "token", "project", 999)
			if !tt.wantErr(t, err) {
				return
			}
			if tt.errIs != nil {
				assert.ErrorIs(t, err, tt.errIs)
			}
		})
	}
}
