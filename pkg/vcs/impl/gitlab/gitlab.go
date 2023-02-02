package gitlab

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"gopkg.in/resty.v1"
)

type GitLab struct {
	Client resty.Client
}

type group struct {
	ID int `json:"id,omitempty"`
}

type project struct {
	ID         int    `json:"id"`
	SshRepoUrl string `json:"ssh_url_to_repo"`
}

type oauthToken struct {
	AccessToken string `json:"access_token"`
}

func (gitlab *GitLab) CheckProjectExist(groupPath, projectName string) (*bool, error) {
	log.Printf("Start check does project already present in group path: %v, by project name: %v...",
		groupPath, projectName)

	projectPath := fmt.Sprintf("%v/%v", groupPath, projectName)

	resp, err := gitlab.Client.R().
		SetQueryParam("simple", "true").
		SetPathParams(map[string]string{
			"project-path": url.PathEscape(projectPath),
		}).
		Get("/api/v4/projects/{project-path}")

	log.Printf("Response received from by GET project request: %v", resp.String())

	if err != nil {
		errorMsg := fmt.Sprintf("failed to check project: %v", err)
		log.Println(errorMsg)

		return nil, errors.New(errorMsg)
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		errorMsg := "unauthorized"
		log.Println(errorMsg)

		return nil, errors.New(errorMsg)
	}

	exist := resp.StatusCode() == http.StatusOK

	return &exist, nil
}

func (gitlab *GitLab) CreateProject(groupPath, projectName string) (string, error) {
	log.Printf("Start creation project by name: %v in group path: %v...", projectName, groupPath)

	id, err := gitlab.GetGroupIdByName(groupPath)

	log.Printf("Id of specified group path: %v is: %v", groupPath, id)

	if err != nil {
		return "", err
	}

	var result project

	resp, err := gitlab.Client.R().
		SetResult(&result).
		SetQueryParams(map[string]string{
			"name":         projectName,
			"namespace_id": id,
		}).
		Post("/api/v4/projects")
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create project in GitLab: %v", err)

		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	if resp.IsError() {
		errorMsg := resp.String()

		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	log.Printf("HTTP Response by create project request: %+v", result)

	return strconv.Itoa(result.ID), nil
}

func (gitlab *GitLab) GetRepositorySshUrl(groupPath, projectName string) (string, error) {
	log.Printf("Start retrieving repository ssh url by group path: %v and project name: %v", groupPath, projectName)

	projectPath := fmt.Sprintf("%v/%v", groupPath, projectName)

	var result project

	resp, err := gitlab.Client.R().
		SetResult(&result).
		SetQueryParam("simple", "true").
		SetPathParams(map[string]string{
			"project-path": url.PathEscape(projectPath),
		}).
		Get("/api/v4/projects/{project-path}")
	if resp.IsError() {
		errorMsg := fmt.Sprintf("Error has received by get project path (%v) request: %v", projectPath, resp.String())
		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	if err != nil {
		errorMsg := fmt.Sprintf("failed get repository SSH URL: %v", err)
		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	log.Printf("HTTP Response by get repository by name: %v is %+v", projectName, result)

	if result.SshRepoUrl != "" {
		log.Printf("SSH URL has been retrieved from the response: %v", result.SshRepoUrl)

		return result.SshRepoUrl, nil
	}

	errMsg := fmt.Sprintf(
		"SSH URL is not presented in the response by group path: %v, project name: %v",
		groupPath, projectName,
	)

	log.Print(errMsg)

	return "", errors.New(errMsg)
}

func (gitlab *GitLab) Init(hostUrl, username, password string) error {
	const maxRetryCount = 3

	log.Printf("Start initialization of username: %v, by url: %v", username, hostUrl)

	client := resty.New()
	client.HostURL = hostUrl

	client.SetRetryCount(maxRetryCount)
	client.AddRetryCondition(
		func(response *resty.Response) (bool, error) {
			return response.IsError(), nil
		},
	)

	token, err := tryToLoginWithPass(hostUrl, username, password)
	if err != nil {
		log.Printf("Error has been occured tring login via password for user: %s", username)

		return fmt.Errorf("failed to login to Gitlab: %w", err)
	}

	log.Printf("Token for username: %v has been retrieved successfully", username)

	client.Token = *token
	gitlab.Client = *client

	return nil
}

func (gitlab *GitLab) GetGroupIdByName(groupName string) (string, error) {
	log.Printf("Start retriving group id by name: %v", groupName)

	var result group

	resp, err := gitlab.Client.R().
		SetQueryParam("simple", "true").
		SetResult(&result).
		SetPathParams(map[string]string{
			"group-name": url.PathEscape(groupName),
		}).
		Get("/api/v4/groups/{group-name}")
	if err != nil {
		errorMsg := fmt.Sprintf("failed get repository group id: %v", err)
		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	if resp.IsError() {
		log.Println(resp.Status())

		return "", errors.New(resp.Status())
	}

	log.Printf("HTTP Response by get group request: %v", result)

	return strconv.Itoa(result.ID), nil
}

func (gitlab *GitLab) DeleteProject(projectId string) error {
	log.Printf("Start project deletion by id: %v", projectId)

	resp, err := gitlab.Client.R().
		SetPathParams(map[string]string{
			"project-id": projectId,
		}).
		Delete("/api/v4/projects/{project-id}")

	log.Printf("Response received from by DELETE project request: %v", resp.String())

	if err != nil {
		errorMsg := fmt.Sprintf("failed to delete project in GitLab: %v", err)
		log.Println(errorMsg)

		return errors.New(errorMsg)
	}

	if resp.IsError() {
		log.Println(resp.Status())

		return errors.New(resp.Status())
	}

	return nil
}

func tryToLoginWithPass(hostUrl, user, pass string) (*string, error) {
	var result oauthToken

	resp, err := resty.R().
		SetResult(&result).
		SetFormData(map[string]string{
			"grant_type": "password",
			"username":   user,
			"password":   pass,
		}).
		Post(hostUrl + "/oauth/token")
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get GitLab access token: %v", err)
		log.Println(errorMsg)

		return nil, errors.New(errorMsg)
	}

	if resp.IsError() {
		errorMsg := resp.String()
		log.Println(errorMsg)

		return nil, errors.New(errorMsg)
	}

	return &result.AccessToken, nil
}
