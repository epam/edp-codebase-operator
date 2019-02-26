package gitlab

import (
	"errors"
	"fmt"
	"gopkg.in/resty.v1"
	"log"
	"net/url"
	"strconv"
)

type GitLab struct {
	Client resty.Client
}

func (gitlab GitLab) CheckProjectExist(groupPath, projectName string) (*bool, error) {
	projectPath := fmt.Sprintf("%v/%v", groupPath, projectName)
	resp, err := gitlab.Client.R().
		SetQueryParam("simple", "true").
		SetPathParams(map[string]string{
			"project-path": url.PathEscape(projectPath),
		}).
		Get("/api/v4/projects/{project-path}")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to check project: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	if resp.StatusCode() == 401 {
		errorMsg := "unauthorized"
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	exist := resp.StatusCode() == 200
	return &exist, nil
}

func (gitlab GitLab) CreateProject(groupPath, projectName string) (string, error) {
	id, err := gitlab.GetGroupIdByName(groupPath)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	resp, err := gitlab.Client.R().
		SetResult(&result).
		SetQueryParams(map[string]string{
			"name":         projectName,
			"namespace_id": id,
		}).
		Post("/api/v4/projects")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to create project in GitLab: %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}
	if resp.IsError() {
		errorMsg := fmt.Sprintf(resp.String())
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}
	return simpleConvertFloatToString(result["id"].(float64)), nil
}

func (gitlab *GitLab) GetRepositorySshUrl(groupPath, projectName string) (string, error) {
	projectPath := fmt.Sprintf("%v/%v", groupPath, projectName)
	var result map[string]interface{}
	_, err := gitlab.Client.R().
		SetResult(&result).
		SetQueryParam("simple", "true").
		SetPathParams(map[string]string{
			"project-path": url.PathEscape(projectPath),
		}).
		Get("/api/v4/projects/{project-path}")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable get repository SSH URL: %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}
	sshUrl := result["ssh_url_to_repo"].(string)

	return sshUrl, nil
}

func (gitlab *GitLab) Init(url string, username string, password string) error {
	client := resty.New()
	client.SetRetryCount(3)
	client.HostURL = url
	client.AddRetryCondition(
		func(response *resty.Response) (bool, error) {
			return response.IsError(), nil
		},
	)

	token, err := tryToLoginWithPass(url, username, password)
	if err != nil {
		log.Printf("Error has been occured tring login via password for user: %s", username)
		log.Printf("Setting private_token for user %s", username)
		client.SetQueryParam("private_token", password)
	} else {
		client.Token = *token
	}

	gitlab.Client = *client
	return nil
}

func tryToLoginWithPass(url, user, pass string) (*string, error) {
	var result map[string]interface{}
	resp, err := resty.R().
		SetResult(&result).
		SetFormData(map[string]string{
			"grant_type": "password",
			"username":   user,
			"password":   pass,
		}).
		Post(url + "/oauth/token")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get GitLab access token: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}

	if resp.IsError() {
		errorMsg := fmt.Sprintf(resp.String())
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	token := result["access_token"].(string)
	return &token, nil
}

func (gitlab GitLab) GetGroupIdByName(groupName string) (string, error) {
	var result map[string]interface{}
	resp, err := gitlab.Client.R().
		SetQueryParam("simple", "true").
		SetResult(&result).
		SetPathParams(map[string]string{
			"group-name": url.PathEscape(groupName),
		}).
		Get("/api/v4/groups/{group-name}")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable get repository group id: %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}
	if resp.IsError() {
		log.Println(resp.Status())
		return "", errors.New(resp.Status())
	}
	return simpleConvertFloatToString(result["id"].(float64)), nil
}

func simpleConvertFloatToString(number float64) string {
	return strconv.FormatFloat(number, 'f', -1, 64)
}

func (gitlab GitLab) DeleteProject(projectId string) error {
	resp, err := gitlab.Client.R().
		SetPathParams(map[string]string{
			"project-id": projectId,
		}).
		Delete("/api/v4/projects/{project-id}")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to delete project in GitLab: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	if resp.IsError() {
		log.Println(resp.Status())
		return errors.New(resp.Status())
	}
	return nil
}
