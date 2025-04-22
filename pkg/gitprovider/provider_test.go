package gitprovider

import (
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestNewProvider(t *testing.T) {
	restyClient := resty.New()

	tests := []struct {
		name        string
		gitServer   *codebaseApi.GitServer
		want        GitWebHookProvider
		wantErr     require.ErrorAssertionFunc
		errContains string
	}{
		{
			name: "github provider",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitProvider: codebaseApi.GitProviderGithub,
				},
			},
			want:    NewGitHubClient(restyClient),
			wantErr: require.NoError,
		},
		{
			name: "gitlab provider",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitProvider: codebaseApi.GitProviderGitlab,
				},
			},
			want:    NewGitLabClient(restyClient),
			wantErr: require.NoError,
		},
		{
			name: "gerrit provider",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitProvider: codebaseApi.GitProviderGerrit,
				},
			},
			want:        nil,
			wantErr:     require.Error,
			errContains: "unsupported git provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProvider(tt.gitServer, restyClient, "")
			tt.wantErr(t, err)

			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_getGitProviderAPIURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		gitServer *codebaseApi.GitServer
		want      string
	}{
		{
			name: "gitlab host",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitHost:     "gitlab.com",
					HttpsPort:   8443,
					GitProvider: codebaseApi.GitProviderGitlab,
				},
			},
			want: "https://gitlab.com:8443",
		},
		{
			name: "github host",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitHost:     "github.com",
					GitProvider: codebaseApi.GitProviderGithub,
				},
			},
			want: "https://api.github.com",
		},
		{
			name: "github Enterprise host",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitHost:     "company.github.com",
					GitProvider: codebaseApi.GitProviderGithub,
				},
			},
			want: "https://company.github.com/api/v3",
		},
		{
			name: "bitbucket host",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					GitProvider: codebaseApi.GitProviderBitbucket,
				},
			},
			want: "https://api.bitbucket.org/2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, GetGitProviderAPIURL(tt.gitServer))
		})
	}
}
