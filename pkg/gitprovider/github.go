package gitprovider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

type gitHubWebHook struct {
	ID     int `json:"id"`
	Config struct {
		URL string `json:"url"`
	} `json:"config"`
}

type GitHubClient struct {
	restyClient *resty.Client
}

// NewGitHubClient creates a new GitHub client.
func NewGitHubClient(restyClient *resty.Client) *GitHubClient {
	restyClient.SetRetryCount(retryCount)
	restyClient.AddRetryCondition(
		func(response *resty.Response, err error) bool {
			return response.IsError()
		},
	)

	return &GitHubClient{restyClient: restyClient}
}

// CreateWebHook creates a new webhook for the given project.
func (c *GitHubClient) CreateWebHook(
	ctx context.Context,
	githubURL,
	token,
	projectID,
	webHookSecret,
	webHookURL string,
) (*WebHook, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}

	c.restyClient.HostURL = githubURL
	webHook := &gitHubWebHook{}

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetBody(map[string]interface{}{
			"name":   "web",
			"active": true,
			"events": []string{"pull_request", "push"},
			"config": map[string]string{
				"url":          webHookURL,
				"content_type": "json",
				"insecure_ssl": "0",
				"secret":       webHookSecret,
			},
		}).
		SetPathParams(map[string]string{
			"owner": owner,
			"repo":  repo,
		}).
		SetResult(webHook).
		Post("/repos/{owner}/{repo}/hooks")

	if err != nil {
		return nil, fmt.Errorf("unable to create GitHub web hook: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("unable to create GitHub web hook: %s", resp.String())
	}

	return convertWebhook(webHook), nil
}

// GetWebHook gets a webhook by ID for the given project.
func (c *GitHubClient) GetWebHook(
	ctx context.Context,
	githubURL,
	token,
	projectID string,
	webHookID int,
) (*WebHook, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}

	c.restyClient.HostURL = githubURL
	webHook := &gitHubWebHook{}

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetPathParams(map[string]string{
			"owner":   owner,
			"repo":    repo,
			"hook-id": strconv.Itoa(webHookID),
		}).
		SetResult(webHook).
		Get("/repos/{owner}/{repo}/hooks/{hook-id}")

	if err != nil {
		return nil, fmt.Errorf("unable to get GitHub web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("unable to get GitHub web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("unable to get GitHub web hook: %s", resp.String())
	}

	return convertWebhook(webHook), nil
}

// DeleteWebHook deletes webhook by ID for the given project.
func (c *GitHubClient) DeleteWebHook(
	ctx context.Context,
	githubURL,
	token,
	projectID string,
	webHookID int,
) error {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return err
	}

	c.restyClient.HostURL = githubURL

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetPathParams(map[string]string{
			"owner":   owner,
			"repo":    repo,
			"hook-id": strconv.Itoa(webHookID),
		}).
		Delete("/repos/{owner}/{repo}/hooks/{hook-id}")

	if err != nil {
		return fmt.Errorf("unable to delete GitHub web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("unable to delete GitHub web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return fmt.Errorf("unable to delete GitHub web hook: %s", resp.String())
	}

	return nil
}

func convertWebhook(githubHook *gitHubWebHook) *WebHook {
	if githubHook == nil {
		return nil
	}

	return &WebHook{
		ID:  githubHook.ID,
		URL: githubHook.Config.URL,
	}
}

func parseProjectID(projectID string) (owner, repo string, err error) {
	parts := strings.Split(projectID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid project ID: %s", projectID)
	}

	return parts[0], parts[1], nil
}
