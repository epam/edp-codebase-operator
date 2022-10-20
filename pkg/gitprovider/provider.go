package gitprovider

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
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
	) (*WebHook, error)
	GetWebHook(
		ctx context.Context,
		gitProviderURL,
		token,
		projectID string,
		webHookID int,
	) (*WebHook, error)
	DeleteWebHook(
		ctx context.Context,
		gitProviderURL,
		token,
		projectID string,
		webHookID int,
	) error
}

// NewProvider creates a new Git web hook provider based on gitServer.
func NewProvider(gitServer *codebaseApi.GitServer, restyClient *resty.Client) (GitWebHookProvider, error) {
	switch gitServer.Spec.GitProvider {
	case codebaseApi.GitProviderGithub:
		return NewGitHubClient(restyClient), nil
	case codebaseApi.GitProviderGitlab:
		return NewGitLabClient(restyClient), nil
	default:
		return nil, fmt.Errorf("unsupported git provider %s", gitServer.Spec.GitProvider)
	}
}
