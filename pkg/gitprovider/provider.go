package gitprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// GitWebHookProvider is an interface for Git web hook provider.
type GitWebHookProvider interface {
	CreateWebHook(
		ctx context.Context,
		gitProviderURL,
		token,
		projectID,
		webHookSecret,
		webHookURL string,
		skipTLS bool,
	) (*WebHook, error)
	CreateWebHookIfNotExists(
		ctx context.Context,
		githubURL,
		token,
		projectID,
		webHookSecret,
		webHookURL string,
		skipTLS bool,
	) (*WebHook, error)
	GetWebHook(
		ctx context.Context,
		gitProviderURL,
		token,
		projectID string,
		webHookRef string,
	) (*WebHook, error)
	GetWebHooks(
		ctx context.Context,
		githubURL,
		token,
		projectID string,
	) ([]*WebHook, error)
	DeleteWebHook(
		ctx context.Context,
		gitProviderURL,
		token,
		projectID string,
		webHookRef string,
	) error
}

// GitProjectProvider is an interface for Git project provider.
type GitProjectProvider interface {
	CreateProject(
		ctx context.Context,
		gitlabURL,
		token,
		fullPath string,
		settings RepositorySettings,
	) error
	ProjectExists(
		ctx context.Context,
		gitlabURL,
		token,
		projectID string,
	) (bool, error)
	SetDefaultBranch(
		ctx context.Context,
		githubURL,
		token,
		projectID,
		branch string,
	) error
}

type RepositorySettings struct {
	IsPrivate bool
}

func (rs RepositorySettings) Visibility() string {
	if rs.IsPrivate {
		return "private"
	}

	return "public"
}

type GitProvider interface {
	GitWebHookProvider
	GitProjectProvider
}

// NewProvider creates a new Git provider based on gitServer.
func NewProvider(gitServer *codebaseApi.GitServer, restyClient *resty.Client, token string) (GitProvider, error) {
	switch gitServer.Spec.GitProvider {
	case codebaseApi.GitProviderGithub:
		return NewGitHubClient(restyClient), nil
	case codebaseApi.GitProviderGitlab:
		return NewGitLabClient(restyClient), nil
	case codebaseApi.GitProviderBitbucket:
		return NewBitbucketClient(token)
	default:
		return nil, fmt.Errorf("unsupported git provider %s", gitServer.Spec.GitProvider)
	}
}

// NewGitProjectProvider creates a new Git project provider based on gitServer.
func NewGitProjectProvider(gitServer *codebaseApi.GitServer, token string) (GitProjectProvider, error) {
	return NewProvider(gitServer, resty.New(), token)
}

// GetGitProviderAPIURL returns git server url with protocol.
func GetGitProviderAPIURL(gitServer *codebaseApi.GitServer) string {
	url := util.GetHostWithProtocol(gitServer.Spec.GitHost)

	if gitServer.Spec.GitProvider == codebaseApi.GitProviderGithub {
		// GitHub API url is different for enterprise and other versions
		// see: https://docs.github.com/en/get-started/learning-about-github/about-versions-of-github-docs#github-enterprise-server
		if url == "https://github.com" {
			return "https://api.github.com"
		}

		url = fmt.Sprintf("%s/api/v3", url)
	}

	if gitServer.Spec.GitProvider == codebaseApi.GitProviderBitbucket {
		return "https://api.bitbucket.org/2.0"
	}

	if gitServer.Spec.HttpsPort != 0 {
		url = fmt.Sprintf("%s:%d", url, gitServer.Spec.HttpsPort)
	}

	return url
}

func parseProjectID(projectID string) (owner, repo string, err error) {
	parts := strings.Split(projectID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid project ID: %s", projectID)
	}

	return parts[0], parts[1], nil
}

type WebHook struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
