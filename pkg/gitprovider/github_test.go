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

func TestGitHubClient_CreateWebHook(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name        string
		projectID   string
		respStatus  int
		resBody     map[string]interface{}
		want        *WebHook
		wantErr     require.ErrorAssertionFunc
		errContains string
	}{
		{
			name:       "success",
			projectID:  "owner/repo",
			respStatus: http.StatusCreated,
			resBody:    map[string]interface{}{"id": 1, "config": map[string]string{"url": "https://example.com"}},
			want:       &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:    require.NoError,
		},
		{
			name:        "response failure",
			projectID:   "owner/repo",
			respStatus:  http.StatusBadRequest,
			resBody:     map[string]interface{}{"message": "bad request"},
			wantErr:     require.Error,
			errContains: "unable to create GitHub web hook",
		},
		{
			name:        "invalid projectID",
			projectID:   "owner-repo",
			respStatus:  http.StatusOK,
			resBody:     map[string]interface{}{"id": 1, "config": map[string]string{"url": "https://example.com"}},
			wantErr:     require.Error,
			errContains: "invalid project ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder, err := httpmock.NewJsonResponder(tt.respStatus, tt.resBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, responder)

			c := NewGitHubClient(restyClient)

			got, err := c.CreateWebHook(context.Background(), "url", "token", tt.projectID, "secret", "webHookURL")

			tt.wantErr(t, err)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitHubClient_GetWebHook(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	fakeUrlRegexp := regexp.MustCompile(`.*`)

	tests := []struct {
		name        string
		projectID   string
		respStatus  int
		resBody     map[string]interface{}
		want        *WebHook
		wantErr     require.ErrorAssertionFunc
		errIs       error
		errContains string
	}{
		{
			name:       "success",
			projectID:  "owner/repo",
			respStatus: http.StatusOK,
			resBody:    map[string]interface{}{"id": 1, "config": map[string]string{"url": "https://example.com"}},
			want:       &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:    require.NoError,
		},
		{
			name:        "invalid project ID",
			projectID:   "owner-repo",
			respStatus:  http.StatusOK,
			resBody:     map[string]interface{}{"id": 1, "config": map[string]string{"url": "https://example.com"}},
			wantErr:     require.Error,
			errContains: "invalid project ID",
		},
		{
			name:        "not found",
			projectID:   "owner/repo",
			respStatus:  http.StatusNotFound,
			resBody:     map[string]interface{}{"message": "not found"},
			wantErr:     require.Error,
			errIs:       ErrWebHookNotFound,
			errContains: "webhook not found",
		},
		{
			name:        "response failure",
			projectID:   "owner/repo",
			respStatus:  http.StatusBadRequest,
			resBody:     map[string]interface{}{"message": "bad request"},
			wantErr:     require.Error,
			errContains: "unable to get GitHub web hook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder, err := httpmock.NewJsonResponder(tt.respStatus, tt.resBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, responder)

			c := NewGitHubClient(restyClient)

			got, err := c.GetWebHook(context.Background(), "url", "token", tt.projectID, 999)

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

func TestGitHubClient_DeleteWebHook(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name        string
		projectID   string
		respStatus  int
		wantErr     require.ErrorAssertionFunc
		errIs       error
		errContains string
	}{
		{
			name:       "success",
			projectID:  "owner/repo",
			respStatus: http.StatusOK,
			wantErr:    require.NoError,
		},
		{
			name:        "not found",
			projectID:   "owner/repo",
			respStatus:  http.StatusNotFound,
			wantErr:     require.Error,
			errIs:       ErrWebHookNotFound,
			errContains: "not found",
		},
		{
			name:        "failure",
			projectID:   "owner/repo",
			respStatus:  http.StatusBadRequest,
			wantErr:     require.Error,
			errContains: "unable to delete GitHub web hook",
		},
		{
			name:        "invalid project ID",
			projectID:   "owner-repo",
			respStatus:  http.StatusOK,
			wantErr:     require.Error,
			errContains: "invalid project ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder := httpmock.NewStringResponder(tt.respStatus, "")
			httpmock.RegisterRegexpResponder(http.MethodDelete, fakeUrlRegexp, responder)

			c := NewGitHubClient(restyClient)

			err := c.DeleteWebHook(context.Background(), "url", "token", tt.projectID, 999)

			tt.wantErr(t, err)
			if tt.errIs != nil {
				assert.ErrorIs(t, err, tt.errIs)
			}
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}

func TestGitHubClient_GetWebHooks(t *testing.T) {
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
			resBody:    []map[string]interface{}{{"id": 1, "config": map[string]string{"url": "https://example.com"}}},
			want:       []*WebHook{{ID: 1, URL: "https://example.com"}},
			wantErr:    require.NoError,
		},
		{
			name:        "invalid project ID",
			projectID:   "owner-repo",
			respStatus:  http.StatusOK,
			resBody:     []map[string]interface{}{{"id": 1, "config": map[string]string{"url": "https://example.com"}}},
			wantErr:     require.Error,
			errContains: "invalid project ID",
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
			errContains: "unable to get GitHub web hooks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			responder, err := httpmock.NewJsonResponder(tt.respStatus, tt.resBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, responder)

			c := NewGitHubClient(restyClient)

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

func TestGitHubClient_CreateWebHookIfNotExists(t *testing.T) {
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
			POSTResBody:    map[string]interface{}{"id": 1, "config": map[string]string{"url": "https://example.com"}},
			want:           &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:           "success - use already existing web hook",
			projectID:      "owner/repo",
			webHookURL:     "https://example.com",
			GETRespStatus:  http.StatusOK,
			GETResBody:     []map[string]interface{}{{"id": 1, "config": map[string]string{"url": "https://example.com"}}},
			POSTRespStatus: http.StatusCreated,
			POSTResBody:    map[string]interface{}{"id": 2, "config": map[string]string{"url": "https://example.com"}},
			want:           &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:           "success - create new web hook with different URL",
			projectID:      "owner/repo",
			webHookURL:     "https://example.com",
			GETRespStatus:  http.StatusOK,
			GETResBody:     []map[string]interface{}{{"id": 2, "config": map[string]string{"url": "https://provider.com"}}},
			POSTRespStatus: http.StatusCreated,
			POSTResBody:    map[string]interface{}{"id": 1, "config": map[string]string{"url": "https://example.com"}},
			want:           &WebHook{ID: 1, URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:          "get webhooks response failure",
			projectID:     "owner/repo",
			GETRespStatus: http.StatusBadRequest,
			GETResBody:    map[string]interface{}{"message": "bad request"},
			wantErr:       require.Error,
			errContains:   "unable to get GitHub web hooks",
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
		{
			name:        "invalid projectID",
			projectID:   "owner-repo",
			wantErr:     require.Error,
			errContains: "invalid project ID",
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

			c := NewGitHubClient(restyClient)

			got, err := c.CreateWebHookIfNotExists(context.Background(), "url", "token", tt.projectID, "secret", tt.webHookURL)

			tt.wantErr(t, err)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
