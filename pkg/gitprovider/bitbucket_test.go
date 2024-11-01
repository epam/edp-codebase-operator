package gitprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitbucketClient_CreateWebHook(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "repo/success") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid": "123", "url": "https://example.com"}`))

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(server.Close)

	tests := []struct {
		name      string
		projectID string
		want      *WebHook
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "create webhook success",
			projectID: "repo/success",
			want: &WebHook{
				ID:  "123",
				URL: "https://example.com",
			},
			wantErr: require.NoError,
		},
		{
			name:      "failed to create webhook",
			projectID: "repo/error",
			want:      nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create Bitbucket web hook")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token", WithBitbucketClientUrl(server.URL))
			require.NoError(t, err)

			got, err := b.CreateWebHook(
				context.Background(),
				"",
				"",
				tt.projectID,
				"secret",
				"webhook",
				true,
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBitbucketClient_GetWebHook(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "123") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"uuid": "123", "url": "https://example.com"}`))

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(server.Close)

	tests := []struct {
		name       string
		webHookRef string
		want       *WebHook
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name:       "get webhook success",
			webHookRef: "123",
			want: &WebHook{
				ID:  "123",
				URL: "https://example.com",
			},
			wantErr: require.NoError,
		},
		{
			name:       "failed to get webhook",
			webHookRef: "000",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get Bitbucket web hook")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token", WithBitbucketClientUrl(server.URL))
			require.NoError(t, err)

			got, err := b.GetWebHook(
				context.Background(),
				"",
				"",
				"owner/repo",
				tt.webHookRef,
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBitbucketClient_CreateWebHookIfNotExists(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if strings.Contains(r.URL.Path, "repo/success") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"values": [{"uuid": "123", "url": "https://example.com"}]}`))

				return
			}
		}

		if r.Method == http.MethodPost {
			if strings.Contains(r.URL.Path, "repo/success") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"uuid": "123", "url": "create-webhook"}`))

				return
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(server.Close)

	tests := []struct {
		name       string
		projectID  string
		webHookURL string
		want       *WebHook
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name:       "create webhook if not exists success",
			projectID:  "repo/success",
			webHookURL: "webhook",
			want: &WebHook{
				ID:  "123",
				URL: "create-webhook",
			},
			wantErr: assert.NoError,
		},
		{
			name:       "webhook already exists",
			projectID:  "repo/success",
			webHookURL: "https://example.com",
			want: &WebHook{
				ID:  "123",
				URL: "https://example.com",
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token", WithBitbucketClientUrl(server.URL))
			require.NoError(t, err)

			got, err := b.CreateWebHookIfNotExists(
				context.Background(),
				"",
				"",
				tt.projectID,
				"secret",
				tt.webHookURL,
				true,
			)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBitbucketClient_DeleteWebHook(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "success") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)

			return
		}

		if strings.Contains(r.URL.Path, "not-found") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)

			_, _ = w.Write([]byte(`{"type": "error", "error": {"message": "Not Found"}}`))

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(server.Close)

	tests := []struct {
		name       string
		webHookRef string
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name:       "delete webhook success",
			webHookRef: "success",
			wantErr:    require.NoError,
		},
		{
			name:       "webhook not found",
			webHookRef: "not-found",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrWebHookNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token", WithBitbucketClientUrl(server.URL))
			require.NoError(t, err)

			tt.wantErr(t, b.DeleteWebHook(
				context.Background(),
				"",
				"",
				"owner/repo",
				tt.webHookRef,
			))
		})
	}
}

func TestBitbucketClient_CreateProject(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "repo/success") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(server.Close)

	tests := []struct {
		name      string
		projectID string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "create project success",
			projectID: "repo/success",
			wantErr:   require.NoError,
		},
		{
			name:      "failed to create project",
			projectID: "repo/error",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create Bitbucket repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token", WithBitbucketClientUrl(server.URL))
			require.NoError(t, err)

			tt.wantErr(t, b.CreateProject(context.Background(), "", "", tt.projectID))
		})
	}
}

func TestBitbucketClient_ProjectExists(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "success") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"values": [{"full_name": "repo/success"}]}`))

			return
		}

		if strings.Contains(r.URL.RawQuery, "empty") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(server.Close)

	tests := []struct {
		name      string
		projectID string
		want      bool
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "project exists",
			projectID: "repo/success",
			want:      true,
			wantErr:   require.NoError,
		},
		{
			name:      "project not found",
			projectID: "repo/success2",
			want:      false,
			wantErr:   require.NoError,
		},
		{
			name:      "failed to get project",
			projectID: "repo/error",
			want:      false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get Bitbucket repository")
			},
		},
		{
			name:      "empty response",
			projectID: "repo/empty",
			want:      false,
			wantErr:   require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token", WithBitbucketClientUrl(server.URL))
			require.NoError(t, err)

			got, err := b.ProjectExists(context.Background(), "", "", tt.projectID)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBitbucketClient_SetDefaultBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		branch  string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:   "set default branch success",
			branch: "new-branch",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrApiNotSupported)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBitbucketClient("token")
			require.NoError(t, err)

			tt.wantErr(t, b.SetDefaultBranch(context.Background(), "", "", "", tt.branch))
		})
	}
}
