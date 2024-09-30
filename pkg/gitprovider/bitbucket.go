// nolint // TODO: add linters after bitbucket implementation
package gitprovider

import (
	"context"
	"errors"
)

type BitbucketClient struct {
}

func NewBitbucketClient() *BitbucketClient {
	return &BitbucketClient{}
}

func (b BitbucketClient) CreateWebHook(ctx context.Context, gitProviderURL, token, projectID, webHookSecret, webHookURL string, skipTLS bool) (*WebHook, error) {
	return nil, errors.New("not implemented")
}

func (b BitbucketClient) CreateWebHookIfNotExists(ctx context.Context, githubURL, token, projectID, webHookSecret, webHookURL string, skipTLS bool) (*WebHook, error) {
	return nil, errors.New("not implemented")
}

func (b BitbucketClient) GetWebHook(ctx context.Context, gitProviderURL, token, projectID string, webHookID int) (*WebHook, error) {
	return nil, errors.New("not implemented")
}

func (b BitbucketClient) GetWebHooks(ctx context.Context, githubURL, token, projectID string) ([]*WebHook, error) {
	return nil, errors.New("not implemented")
}

func (b BitbucketClient) DeleteWebHook(ctx context.Context, gitProviderURL, token, projectID string, webHookID int) error {
	return errors.New("not implemented")
}

func (b BitbucketClient) CreateProject(ctx context.Context, gitlabURL, token, fullPath string) error {
	return errors.New("not implemented")
}

func (b BitbucketClient) ProjectExists(ctx context.Context, gitlabURL, token, projectID string) (bool, error) {
	return false, errors.New("not implemented")
}

func (b BitbucketClient) SetDefaultBranch(ctx context.Context, githubURL, token, projectID, branch string) error {
	return errors.New("not implemented")
}
