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
			want:       &WebHook{ID: "1", URL: "https://example.com"},
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

			got, err := c.CreateWebHook(context.Background(), "url", "token", "project", "secret", "webHookURL", false)
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
			want:       &WebHook{ID: "1", URL: "https://example.com"},
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

			got, err := c.GetWebHook(context.Background(), "url", "token", "project", "999")
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

			err := c.DeleteWebHook(context.Background(), "url", "token", "project", "999")
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
			want:       []*WebHook{{ID: "1", URL: "https://example.com"}},
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
			errContains: "failed to get GitLab web hooks",
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
			want:           &WebHook{ID: "1", URL: "https://example.com"},
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
			want:           &WebHook{ID: "1", URL: "https://example.com"},
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
			want:           &WebHook{ID: "1", URL: "https://example.com"},
			wantErr:        require.NoError,
		},
		{
			name:          "get webhooks response failure",
			projectID:     "owner/repo",
			GETRespStatus: http.StatusBadRequest,
			GETResBody:    map[string]interface{}{"message": "bad request"},
			wantErr:       require.Error,
			errContains:   "failed to get GitLab web hooks",
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

			got, err := c.CreateWebHookIfNotExists(context.Background(), "url", "token", tt.projectID, "secret", tt.webHookURL, false)

			tt.wantErr(t, err)

			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitLabClient_CreateProject(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name                    string
		projectID               string
		getNsRespStatus         int
		getNsResBody            interface{}
		CreateProjectRespStatus int
		CreateProjectResBody    interface{}
		wantErr                 require.ErrorAssertionFunc
	}{
		{
			name:                    "success - create new project",
			projectID:               "namespace/owner/repo",
			getNsRespStatus:         http.StatusOK,
			getNsResBody:            map[string]int{"id": 1},
			CreateProjectRespStatus: http.StatusCreated,
			CreateProjectResBody:    map[string]int{"id": 1},
			wantErr:                 require.NoError,
		},
		{
			name:            "failed to get namespace",
			projectID:       "namespace/owner/repo",
			getNsRespStatus: http.StatusNotFound,
			getNsResBody:    map[string]string{"message": "not found"},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get GitLab namespace")
			},
		},
		{
			name:                    "failed to create project",
			projectID:               "namespace/owner/repo",
			getNsRespStatus:         http.StatusOK,
			getNsResBody:            map[string]int{"id": 1},
			CreateProjectRespStatus: http.StatusBadRequest,
			CreateProjectResBody:    map[string]string{"message": "not found"},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create GitLab project")
			},
		},
		{
			name:      "invalid project ID",
			projectID: "/repo",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid project ID")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			GETResponder, err := httpmock.NewJsonResponder(tt.getNsRespStatus, tt.getNsResBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)

			POSTResponder, err := httpmock.NewJsonResponder(tt.CreateProjectRespStatus, tt.CreateProjectResBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, POSTResponder)

			c := NewGitLabClient(restyClient)

			err = c.CreateProject(context.Background(), "url", "token", tt.projectID, RepositorySettings{})
			tt.wantErr(t, err)
		})
	}
}

func TestGitLabClient_ProjectExists(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name                 string
		projectID            string
		getProjectRespStatus int
		getProjectResBody    interface{}
		want                 bool
		wantErr              require.ErrorAssertionFunc
	}{
		{
			name:                 "success - project exists",
			projectID:            "namespace/owner/repo",
			getProjectRespStatus: http.StatusOK,
			getProjectResBody:    map[string]int{"id": 1},
			want:                 true,
			wantErr:              require.NoError,
		},
		{
			name:                 "success - project does not exist",
			projectID:            "namespace/owner/repo",
			getProjectRespStatus: http.StatusNotFound,
			getProjectResBody:    map[string]string{"message": "not found"},
			want:                 false,
			wantErr:              require.NoError,
		},
		{
			name:                 "failed to get project",
			projectID:            "namespace/owner/repo",
			getProjectRespStatus: http.StatusBadRequest,
			getProjectResBody:    map[string]string{"message": "bad request"},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get GitLab project")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			GETResponder, err := httpmock.NewJsonResponder(tt.getProjectRespStatus, tt.getProjectResBody)
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)

			c := NewGitLabClient(restyClient)

			got, err := c.ProjectExists(context.Background(), "url", "token", tt.projectID)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitLabClient_SetDefaultBranch(t *testing.T) {
	fakeUrlRegexp := regexp.MustCompile(`.*`)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name       string
		projectID  string
		respStatus int
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name:       "success",
			projectID:  "namespace/owner/repo",
			respStatus: http.StatusOK,
			wantErr:    require.NoError,
		},
		{
			name:       "failed to set default branch",
			projectID:  "namespace/owner/repo",
			respStatus: http.StatusBadRequest,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to set GitLab default branch")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			GETResponder, err := httpmock.NewJsonResponder(tt.respStatus, map[string]string{})
			require.NoError(t, err)
			httpmock.RegisterRegexpResponder(http.MethodPut, fakeUrlRegexp, GETResponder)

			c := NewGitLabClient(restyClient)

			err = c.SetDefaultBranch(context.Background(), "url", "token", tt.projectID, "main")
			tt.wantErr(t, err)
		})
	}
}

func Test_decodeProjectID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		projectID     string
		wantNamespace string
		wantPath      string
		wantErr       require.ErrorAssertionFunc
	}{
		{
			name:          "success with namespace",
			projectID:     "namespace/owner/repo",
			wantNamespace: "namespace/owner",
			wantPath:      "repo",
			wantErr:       require.NoError,
		},
		{
			name:          "success with owner",
			projectID:     "owner/repo",
			wantNamespace: "owner",
			wantPath:      "repo",
			wantErr:       require.NoError,
		},
		{
			name:      "failed - no repo",
			projectID: "namespace",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid project ID")
			},
		},
		{
			name:      "failed - starts with /",
			projectID: "/namespace",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid project ID")
			},
		},
		{
			name:      "failed - ends with /",
			projectID: "namespace/",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid project ID")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNamespace, gotPath, err := decodeProjectID(tt.projectID)
			tt.wantErr(t, err)

			assert.Equal(t, tt.wantNamespace, gotNamespace)
			assert.Equal(t, tt.wantPath, gotPath)
		})
	}
}
