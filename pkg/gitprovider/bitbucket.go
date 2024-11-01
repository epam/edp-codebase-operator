package gitprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider/bitbucket/generated"
)

type BitbucketClient struct {
	client generated.ClientWithResponsesInterface
}

type BitbucketClientOpts struct {
	Url string
}

type BitbucketClientOptsSetter func(*BitbucketClientOpts)

func WithBitbucketClientUrl(url string) func(*BitbucketClientOpts) {
	return func(opts *BitbucketClientOpts) {
		opts.Url = url
	}
}

func NewBitbucketClient(token string, opts ...BitbucketClientOptsSetter) (*BitbucketClient, error) {
	defaults := &BitbucketClientOpts{
		Url: "https://api.bitbucket.org/2.0",
	}

	for _, o := range opts {
		o(defaults)
	}

	tokenProvider := NewBasicTokenAuthProvider(token)

	c, err := generated.NewClientWithResponses(
		defaults.Url,
		generated.WithRequestEditorFn(tokenProvider.Intercept),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bitbucket client: %w", err)
	}

	return &BitbucketClient{
		client: c,
	}, nil
}

func (b *BitbucketClient) CreateWebHook(ctx context.Context, _, _, projectID, webHookSecret, webHookURL string, skipTLS bool) (*WebHook, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}

	r, err := b.client.PostRepositoriesWorkspaceRepoSlugHooksWithResponse(
		ctx,
		owner,
		repo,
		func(ctx context.Context, req *http.Request) error {
			var body []byte

			body, err = json.Marshal(map[string]interface{}{
				"description":            fmt.Sprintf("Automatically created %s", uuid.NewUUID()),
				"url":                    webHookURL,
				"active":                 true,
				"secret":                 webHookSecret,
				"skip_cert_verification": skipTLS,
				"history_enabled":        true,
				"events": []string{
					"pullrequest:created",
					"pullrequest:updated",
					"pullrequest:fulfilled",
					"pullrequest:comment_created",
					"pullrequest:comment_updated",
				},
			})
			if err != nil {
				return fmt.Errorf("failed to marshal Bitbucket web hook body: %w", err)
			}

			req.Body = io.NopCloser(bytes.NewReader(body))

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create Bitbucket web hook: %w", err)
	}

	if !createObjectStatusOk(r.StatusCode()) {
		return nil, fmt.Errorf("failed to create Bitbucket web hook: %s %s", r.Status(), r.Body)
	}

	if r.JSON201 == nil || r.JSON201.Uuid == nil || r.JSON201.Url == nil {
		return nil, fmt.Errorf("failed to create Bitbucket web hook: invalid response %s", r.Body)
	}

	return &WebHook{
		ID:  *r.JSON201.Uuid,
		URL: *r.JSON201.Url,
	}, nil
}

func (b *BitbucketClient) CreateWebHookIfNotExists(ctx context.Context, _, _, projectID, webHookSecret, webHookURL string, skipTLS bool) (*WebHook, error) {
	webHooks, err := b.GetWebHooks(ctx, "", "", projectID)
	if err != nil {
		return nil, err
	}

	for _, webHook := range webHooks {
		if webHook.URL == webHookURL {
			return webHook, nil
		}
	}

	return b.CreateWebHook(ctx, "", "", projectID, webHookSecret, webHookURL, skipTLS)
}

func (b *BitbucketClient) GetWebHook(ctx context.Context, _, _, projectID, webHookRef string) (*WebHook, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}

	r, err := b.client.GetRepositoriesWorkspaceRepoSlugHooksUidWithResponse(ctx, owner, repo, webHookRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bitbucket web hook: %w", err)
	}

	if r.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get Bitbucket web hook: %s %s", r.Status(), r.Body)
	}

	if r.JSON200 == nil || r.JSON200.Uuid == nil || r.JSON200.Url == nil {
		return nil, fmt.Errorf("failed to get Bitbucket web hook: invalid response %s", r.Body)
	}

	return &WebHook{
		ID:  *r.JSON200.Uuid,
		URL: *r.JSON200.Url,
	}, nil
}

func (b *BitbucketClient) GetWebHooks(ctx context.Context, _, _, projectID string) ([]*WebHook, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}

	r, err := b.client.GetRepositoriesWorkspaceRepoSlugHooksWithResponse(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bitbucket web hooks: %w", err)
	}

	if r.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get Bitbucket web hooks: %s %s", r.Status(), r.Body)
	}

	if r.JSON200 == nil || r.JSON200.Values == nil {
		return []*WebHook{}, nil
	}

	webHooks := make([]*WebHook, len(*r.JSON200.Values))

	for i, hook := range *r.JSON200.Values {
		if hook.Uuid == nil || hook.Url == nil {
			return nil, fmt.Errorf("failed to get Bitbucket web hooks: invalid response %s", r.Body)
		}

		webHooks[i] = &WebHook{
			ID:  *hook.Uuid,
			URL: *hook.Url,
		}
	}

	return webHooks, nil
}

func (b *BitbucketClient) DeleteWebHook(ctx context.Context, _, _, projectID, webHookRef string) error {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return err
	}

	r, err := b.client.DeleteRepositoriesWorkspaceRepoSlugHooksUidWithResponse(ctx, owner, repo, webHookRef)
	if err != nil {
		return fmt.Errorf("failed to delete Bitbucket web hook: %w", err)
	}

	if r.StatusCode() != http.StatusNoContent {
		if r.StatusCode() == http.StatusNotFound {
			return fmt.Errorf("failed to delete Bitbucket web hook: %w", ErrWebHookNotFound)
		}

		return fmt.Errorf("failed to delete Bitbucket web hook: %s %s", r.Status(), r.Body)
	}

	return nil
}

func (b *BitbucketClient) CreateProject(ctx context.Context, _, _, projectID string) error {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return err
	}

	reqBody := generated.PostRepositoriesWorkspaceRepoSlugJSONRequestBody{
		Type:      "repository",
		IsPrivate: ptr.To(true),
	}

	r, err := b.client.PostRepositoriesWorkspaceRepoSlugWithResponse(ctx, owner, repo, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create Bitbucket repository: %w", err)
	}

	if !createObjectStatusOk(r.StatusCode()) {
		return fmt.Errorf("failed to create Bitbucket repository: %s %s", r.Status(), r.Body)
	}

	return nil
}

func (b *BitbucketClient) ProjectExists(ctx context.Context, _, _, projectID string) (bool, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return false, err
	}

	r, err := b.client.GetRepositoriesWorkspaceWithResponse(
		ctx,
		owner,
		nil,
		func(ctx context.Context, req *http.Request) error {
			// nolint: gocritic // Can't use %q instead of "%s" because need double quotes.
			req.URL.RawQuery = fmt.Sprintf(`q=slug="%s"`, repo)

			return nil
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to get Bitbucket repository: %w", err)
	}

	if r.StatusCode() != http.StatusOK {
		return false, fmt.Errorf("failed to get Bitbucket repository: %s %s", r.Status(), r.Body)
	}

	if r.JSON200 == nil || r.JSON200.Values == nil {
		return false, nil
	}

	// nolint: gocritic // Force to use copy value.
	for _, v := range *r.JSON200.Values {
		if v.FullName != nil && *v.FullName == projectID {
			return true, nil
		}
	}

	return false, nil
}

func (*BitbucketClient) SetDefaultBranch(_ context.Context, _, _, _, _ string) error {
	// Set default branch is not supported by Bitbucket API.
	// Open ticket: https://jira.atlassian.com/browse/BCLOUD-20340
	// https://community.atlassian.com/t5/Bitbucket-questions/Get-and-set-default-branch-in-v2-API/qaq-p/2416227
	return fmt.Errorf("setting default branch in Bitbucket repository: %w", ErrApiNotSupported)
}

func createObjectStatusOk(statusCode int) bool {
	return statusCode == http.StatusOK || statusCode == http.StatusCreated
}
