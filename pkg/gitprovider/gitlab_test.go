package gitprovider

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

func TestGitLabClient_GetWebHooks(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	fakeUrlRegexp := regexp.MustCompile(`.*`)

	tests := []struct {
		name        string
		projectID   string
		respStatus  int
		resBody     interface{}
		want        []*WebHook
		wantErr     require.ErrorAssertionFunc
		errIs       error
		errContains string
	}{
		{
			name:       "success",
			projectID:  "owner/repo",
			respStatus: http.StatusOK,
			resBody:    []map[string]interface{}{{"id": 1, "url": "https://example.com"}},
			want:       []*WebHook{{ID: 1, URL: "https://example.com"}},
			wantErr:    require.NoError,
		},
		{
			name:       "empty response",
			projectID:  "owner/repo",
			respStatus: http.StatusOK,
			resBody:    []map[string]interface{}{},
			want:       []*WebHook{},
			wantErr:    require.NoError,
		},
		{
			name:        "response failure",
			projectID:   "owner/repo",
			respStatus:  http.StatusBadRequest,
			resBody:     map[string]interface{}{"message": "bad request"},
			wantErr:     require.Error,
			errContains: "unable to get GitLab web hooks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder, err := httpmock.NewJsonResponder(tt.respStatus, tt.resBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, responder)

			c := NewGitLabClient(restyClient)

			got, err := c.GetWebHooks(context.Background(), "url", "token", tt.projectID)

			tt.wantErr(t, err)
			if tt.errIs != nil {
				assert.ErrorIs(t, err, tt.errIs)
			}
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitLabClient_CreateWebHookIfNotExists(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name           string
		projectID      string
		webHookURL     string
		GETRespStatus  int
		GETResBody     interface{}
		POSTRespStatus int
		POSTResBody    interface{}
		want           *WebHook
		wantErr        require.ErrorAssertionFunc
		errContains    string
	}{
		{
			name:           "success - create new web hook",
			projectID:      "owner/repo",
			GETRespStatus:  http.StatusOK,
			GETResBody:     []map[string]interface{}{},
			POSTRespStatus: http.StatusCreated,
			POSTResBody:    map[string]interface{}{"id": 1, "url": "https://example.com"},
			want:           &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:           "success - use already existing web hook",
			projectID:      "owner/repo",
			webHookURL:     "https://example.com",
			GETRespStatus:  http.StatusOK,
			GETResBody:     []map[string]interface{}{{"id": 1, "url": "https://example.com"}},
			POSTRespStatus: http.StatusCreated,
			POSTResBody:    map[string]interface{}{"id": 2, "url": "https://provider.com"},
			want:           &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:           "success - create new web hook with different URL",
			projectID:      "owner/repo",
			GETRespStatus:  http.StatusOK,
			webHookURL:     "https://example.com",
			GETResBody:     []map[string]interface{}{{"id": 2, "url": "https://provider.com"}},
			POSTRespStatus: http.StatusCreated,
			POSTResBody:    map[string]interface{}{"id": 1, "url": "https://example.com"},
			want:           &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:          "get webhooks response failure",
			projectID:     "owner/repo",
			GETRespStatus: http.StatusBadRequest,
			GETResBody:    map[string]interface{}{"message": "bad request"},
			wantErr:       require.Error,
			errContains:   "unable to get GitLab web hooks",
		},
		{
			name:           "create webhooks response failure",
			projectID:      "owner/repo",
			GETRespStatus:  http.StatusOK,
			GETResBody:     []map[string]interface{}{},
			POSTRespStatus: http.StatusBadRequest,
			POSTResBody:    map[string]interface{}{"message": "bad request"},
			wantErr:        require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			GETResponder, err := httpmock.NewJsonResponder(tt.GETRespStatus, tt.GETResBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)

			POSTResponder, err := httpmock.NewJsonResponder(tt.POSTRespStatus, tt.POSTResBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, POSTResponder)

			c := NewGitLabClient(restyClient)

			got, err := c.CreateWebHookIfNotExists(context.Background(), "url", "token", tt.projectID, "secret", tt.webHookURL)

			tt.wantErr(t, err)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
