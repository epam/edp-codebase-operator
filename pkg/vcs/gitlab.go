package vcs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"
)

var ErrWebHookNotFound = errors.New("webhook not found")

type WebHook struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type GitLabClient struct {
	restyClient *resty.Client
}

const retryCount = 3

// NewGitLabClient creates a new GitLab client.
func NewGitLabClient(restyClient *resty.Client) *GitLabClient {
	restyClient.SetRetryCount(retryCount)
	restyClient.AddRetryCondition(
		func(response *resty.Response, err error) bool {
			return response.IsError()
		},
	)

	return &GitLabClient{restyClient: restyClient}
}

// CreateWebHook creates a new webhook for the given project.
func (c *GitLabClient) CreateWebHook(
	ctx context.Context,
	gitlabURL,
	token,
	projectID,
	webHookSecret,
	webHookURL string,
) (*WebHook, error) {
	c.restyClient.HostURL = gitlabURL
	webHook := &WebHook{}

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader("PRIVATE-TOKEN", token).
		SetBody(map[string]interface{}{
			"url":                   webHookURL,
			"merge_requests_events": true,
			"push_events":           false,
			"token":                 webHookSecret,
		}).
		SetPathParams(map[string]string{
			"project-id": projectID,
		}).
		SetResult(webHook).
		Post("/api/v4/projects/{project-id}/hooks")

	if err != nil {
		return nil, fmt.Errorf("unable to create GitLab web hook: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("unable to create GitLab web hook: %s", resp.String())
	}

	return webHook, nil
}

// GetWebHook gets a webhook by ID for the given project.
func (c *GitLabClient) GetWebHook(
	ctx context.Context,
	gitlabURL,
	token,
	projectID string,
	webHookID int,
) (*WebHook, error) {
	c.restyClient.HostURL = gitlabURL
	webHook := &WebHook{}

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader("PRIVATE-TOKEN", token).
		SetPathParams(map[string]string{
			"project-id": projectID,
			"hook-id":    strconv.Itoa(webHookID),
		}).
		SetResult(webHook).
		Get("/api/v4/projects/{project-id}/hooks/{hook-id}")

	if err != nil {
		return nil, fmt.Errorf("unable to get GitLab web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("unable to get GitLab web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("unable to get GitLab web hook: %s", resp.String())
	}

	return webHook, nil
}

// DeleteWebHook deletes webhook by ID for the given project.
func (c *GitLabClient) DeleteWebHook(
	ctx context.Context,
	gitlabURL,
	token,
	projectID string,
	webHookID int,
) error {
	c.restyClient.HostURL = gitlabURL

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader("PRIVATE-TOKEN", token).
		SetPathParams(map[string]string{
			"project-id": projectID,
			"hook-id":    strconv.Itoa(webHookID),
		}).
		Delete("/api/v4/projects/{project-id}/hooks/{hook-id}")

	if err != nil {
		return fmt.Errorf("unable to delete GitLab web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("unable to delete GitLab web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return fmt.Errorf("unable to delete GitLab web hook: %s", resp.String())
	}

	return nil
}
