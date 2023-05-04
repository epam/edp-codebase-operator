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

type gitHubOrganization struct {
	Login string `json:"login"`
}

type GitHubClient struct {
	restyClient *resty.Client
}

const (
	repoPathParam  = "repo"
	ownerPathParam = "owner"
)

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
			"events": []string{"pull_request", "push", "issue_comment"},
			"config": map[string]string{
				"url":          webHookURL,
				"content_type": "json",
				"insecure_ssl": "0",
				"secret":       webHookSecret,
			},
		}).
		SetPathParams(map[string]string{
			ownerPathParam: owner,
			repoPathParam:  repo,
		}).
		SetResult(webHook).
		Post("/repos/{owner}/{repo}/hooks")
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub web hook: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to create GitHub web hook: %s", resp.String())
	}

	return convertWebhook(webHook), nil
}

// CreateWebHookIfNotExists checks if a webhook with a given URL exists in the project.
// If a webhook exists function returns it. If not, creates a new one.
func (c *GitHubClient) CreateWebHookIfNotExists(
	ctx context.Context,
	githubURL,
	token,
	projectID,
	webHookSecret,
	webHookURL string,
) (*WebHook, error) {
	webHooks, err := c.GetWebHooks(ctx, githubURL, token, projectID)
	if err != nil {
		return nil, err
	}

	for _, webHook := range webHooks {
		if webHook.URL == webHookURL {
			return webHook, nil
		}
	}

	return c.CreateWebHook(ctx, githubURL, token, projectID, webHookSecret, webHookURL)
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
			ownerPathParam: owner,
			repoPathParam:  repo,
			"hook-id":      strconv.Itoa(webHookID),
		}).
		SetResult(webHook).
		Get("/repos/{owner}/{repo}/hooks/{hook-id}")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("failed to get GitHub web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get GitHub web hook: %s", resp.String())
	}

	return convertWebhook(webHook), nil
}

// GetWebHooks gets a webhooks by the given project.
func (c *GitHubClient) GetWebHooks(
	ctx context.Context,
	githubURL,
	token,
	projectID string,
) ([]*WebHook, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}

	c.restyClient.HostURL = githubURL

	var gitHubWebHooks []*gitHubWebHook

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetPathParams(map[string]string{
			ownerPathParam: owner,
			repoPathParam:  repo,
		}).
		SetResult(&gitHubWebHooks).
		Get("/repos/{owner}/{repo}/hooks")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub web hooks: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get GitHub web hooks: %s", resp.String())
	}

	webHooks := make([]*WebHook, len(gitHubWebHooks))
	for i, webHook := range gitHubWebHooks {
		webHooks[i] = convertWebhook(webHook)
	}

	return webHooks, nil
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
			ownerPathParam: owner,
			repoPathParam:  repo,
			"hook-id":      strconv.Itoa(webHookID),
		}).
		Delete("/repos/{owner}/{repo}/hooks/{hook-id}")
	if err != nil {
		return fmt.Errorf("failed to delete GitHub web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("failed to delete GitHub web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to delete GitHub web hook: %s", resp.String())
	}

	return nil
}

// CreateProject creates a new project.
func (c *GitHubClient) CreateProject(
	ctx context.Context,
	githubURL,
	token,
	projectID string,
) error {
	c.restyClient.HostURL = githubURL
	path := "user/repos"

	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return err
	}

	isOrg, err := c.isOwnerOrg(ctx, githubURL, token, owner)
	if err != nil {
		return err
	}

	if isOrg {
		path = fmt.Sprintf("orgs/%v/repos", owner)
	}

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetBody(map[string]string{
			"name": repo,
		}).
		Post(path)
	if err != nil {
		return fmt.Errorf("failed to create GitHub repository: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to create GitHub repository: %s", resp.String())
	}

	return nil
}

// ProjectExists checks if the given project exists.
func (c *GitHubClient) ProjectExists(
	ctx context.Context,
	githubURL,
	token,
	projectID string,
) (bool, error) {
	owner, repo, err := parseProjectID(projectID)
	if err != nil {
		return false, err
	}

	c.restyClient.HostURL = githubURL

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetPathParams(map[string]string{
			ownerPathParam: owner,
			repoPathParam:  repo,
		}).
		Get("/repos/{owner}/{repo}")
	if err != nil {
		return false, fmt.Errorf("failed to get GitHub repository: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get GitHub repository: %s", resp.String())
	}

	return true, nil
}

// isOwnerOrg checks if the given owner is an organization.
func (c *GitHubClient) isOwnerOrg(
	ctx context.Context,
	githubURL,
	token string,
	owner string,
) (bool, error) {
	c.restyClient.HostURL = githubURL

	orgs := make([]gitHubOrganization, 0)

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetAuthToken(token).
		SetQueryParam("per_page", "1000").
		SetResult(&orgs).
		Get("/user/orgs")
	if err != nil {
		return false, fmt.Errorf("failed to get GitHub organizations: %w", err)
	}

	if resp.IsError() {
		return false, fmt.Errorf("failed to get GitHub organizations: %s", resp.String())
	}

	for _, org := range orgs {
		if strings.EqualFold(org.Login, owner) {
			return true, nil
		}
	}

	return false, nil
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
