package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func Test_getGitProviderAPIURL(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getGitProviderAPIURL(tt.gitServer))
		})
	}
}
