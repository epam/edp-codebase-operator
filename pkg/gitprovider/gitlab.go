package gitprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

var ErrWebHookNotFound = errors.New("webhook not found")

type WebHook struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type gitlabNamespace struct {
	ID int `json:"id"`
}

type GitLabClient struct {
	restyClient *resty.Client
}

const (
	retryCount            = 3
	gitLabTokenHeaderName = "PRIVATE-TOKEN"
	projectIDPathParam    = "project-id"
)

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
		SetHeader(gitLabTokenHeaderName, token).
		SetBody(map[string]interface{}{
			"url":                   webHookURL,
			"merge_requests_events": true,
			"note_events":           true,
			"push_events":           false,
			"token":                 webHookSecret,
		}).
		SetPathParams(map[string]string{
			projectIDPathParam: projectID,
		}).
		SetResult(webHook).
		Post("/api/v4/projects/{project-id}/hooks")
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab web hook: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to create GitLab web hook: %s", resp.String())
	}

	return webHook, nil
}

// CreateWebHookIfNotExists checks if a webhook with a given URL exists in the project.
// If a webhook exists function returns it. If not, creates a new one.
func (c *GitLabClient) CreateWebHookIfNotExists(
	ctx context.Context,
	gitlabURL,
	token,
	projectID,
	webHookSecret,
	webHookURL string,
) (*WebHook, error) {
	webHooks, err := c.GetWebHooks(ctx, gitlabURL, token, projectID)
	if err != nil {
		return nil, err
	}

	for _, webHook := range webHooks {
		if webHook.URL == webHookURL {
			return webHook, nil
		}
	}

	return c.CreateWebHook(ctx, gitlabURL, token, projectID, webHookSecret, webHookURL)
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
		SetHeader(gitLabTokenHeaderName, token).
		SetPathParams(map[string]string{
			projectIDPathParam: projectID,
			"hook-id":          strconv.Itoa(webHookID),
		}).
		SetResult(webHook).
		Get("/api/v4/projects/{project-id}/hooks/{hook-id}")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitLab web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("failed to get GitLab web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get GitLab web hook: %s", resp.String())
	}

	return webHook, nil
}

// GetWebHooks gets a webhook by the given project.
func (c *GitLabClient) GetWebHooks(
	ctx context.Context,
	gitlabURL,
	token,
	projectID string,
) ([]*WebHook, error) {
	c.restyClient.HostURL = gitlabURL

	var webHooks []*WebHook

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader(gitLabTokenHeaderName, token).
		SetPathParams(map[string]string{
			projectIDPathParam: projectID,
		}).
		SetResult(&webHooks).
		Get("/api/v4/projects/{project-id}/hooks")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitLab web hooks: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get GitLab web hooks: %s", resp.String())
	}

	return webHooks, nil
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
		SetHeader(gitLabTokenHeaderName, token).
		SetPathParams(map[string]string{
			projectIDPathParam: projectID,
			"hook-id":          strconv.Itoa(webHookID),
		}).
		Delete("/api/v4/projects/{project-id}/hooks/{hook-id}")
	if err != nil {
		return fmt.Errorf("failed to delete GitLab web hook: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("failed to delete GitLab web hook: %w", ErrWebHookNotFound)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to delete GitLab web hook: %s", resp.String())
	}

	return nil
}

// CreateProject creates a new project in GitLab.
func (c *GitLabClient) CreateProject(
	ctx context.Context,
	gitlabURL,
	token,
	projectID string,
) error {
	namespace, path, err := decodeProjectID(projectID)
	if err != nil {
		return err
	}

	gitLabNs, err := c.getNamespace(ctx, gitlabURL, token, namespace)
	if err != nil {
		return err
	}

	c.restyClient.HostURL = gitlabURL
	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader(gitLabTokenHeaderName, token).
		SetBody(map[string]any{
			"path":         path,
			"namespace_id": gitLabNs.ID,
		}).
		Post("/api/v4/projects")

	if err != nil {
		return fmt.Errorf("failed to create GitLab project: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to create GitLab project: %s", resp.String())
	}

	return nil
}

// ProjectExists checks if a project exists.
func (c *GitLabClient) ProjectExists(
	ctx context.Context,
	gitlabURL,
	token,
	projectID string,
) (bool, error) {
	c.restyClient.HostURL = gitlabURL
	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader(gitLabTokenHeaderName, token).
		SetPathParams(map[string]string{
			"projectID": projectID,
		}).
		Get("/api/v4/projects/{projectID}")

	if err != nil {
		return false, fmt.Errorf("failed to check if GitLab project exists: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get GitLab project: %s", resp.String())
	}

	return true, nil
}

func (c *GitLabClient) getNamespace(
	ctx context.Context,
	gitlabURL,
	token,
	namespaceFullPath string,
) (*gitlabNamespace, error) {
	c.restyClient.HostURL = gitlabURL
	ns := &gitlabNamespace{}

	resp, err := c.restyClient.
		R().
		SetContext(ctx).
		SetHeader(gitLabTokenHeaderName, token).
		SetPathParams(map[string]string{
			"gitlabNamespace": namespaceFullPath,
		}).
		SetResult(ns).
		Get("/api/v4/namespaces/{gitlabNamespace}")

	if err != nil {
		return nil, fmt.Errorf("failed to get GitLab namespace: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get GitLab namespace: %s", resp.String())
	}

	return ns, nil
}

func decodeProjectID(projectID string) (namespace, path string, err error) {
	lastSlashIndex := strings.LastIndex(projectID, "/")
	if lastSlashIndex == -1 || lastSlashIndex == len(projectID)-1 || lastSlashIndex == 0 {
		return "", "", fmt.Errorf("invalid project ID: %s", projectID)
	}

	return projectID[:lastSlashIndex], projectID[lastSlashIndex+1:], nil
}
