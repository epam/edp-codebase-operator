package gitprovider

import (
	"context"
	"fmt"

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
		webHookID int,
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
		webHookID int,
	) error
}

// GitProjectProvider is an interface for Git project provider.
type GitProjectProvider interface {
	CreateProject(
		ctx context.Context,
		gitlabURL,
		token,
		fullPath string,
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

type GitProvider interface {
	GitWebHookProvider
	GitProjectProvider
}

// NewProvider creates a new Git provider based on gitServer.
func NewProvider(gitServer *codebaseApi.GitServer, restyClient *resty.Client) (GitProvider, error) {
	switch gitServer.Spec.GitProvider {
	case codebaseApi.GitProviderGithub:
		return NewMockGitHubClient(restyClient), nil
	case codebaseApi.GitProviderGitlab:
		return NewMockGitLabClient(restyClient), nil
	default:
		return nil, fmt.Errorf("unsupported git provider %s", gitServer.Spec.GitProvider)
	}
}

// NewMockGitProjectProvider creates a new Git project provider based on gitServer.
func NewMockGitProjectProvider(gitServer *codebaseApi.GitServer) (GitProjectProvider, error) {
	return NewProvider(gitServer, resty.New())
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

	if gitServer.Spec.HttpsPort != 0 {
		url = fmt.Sprintf("%s:%d", url, gitServer.Spec.HttpsPort)
	}

	return url
}
