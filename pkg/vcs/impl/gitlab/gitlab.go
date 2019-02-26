package gitlab

import (
	"encoding/json"
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

func (gitlab *GitLab) Init(url string, username string, password string) error {
	client := resty.New()
	client.SetRetryCount(3)
	client.HostURL = url
	client.AddRetryCondition(
		func(response *resty.Response) (bool, error) {
			return response.IsError(), nil
		},
	)

	resp, err := resty.R().
		SetFormData(map[string]string{
			"grant_type": "password",
			"username":   username,
			"password":   password,
		}).
		Post(url + "/oauth/token")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get GitLab access token: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}

	if resp.IsError() {
		errorMsg := fmt.Sprintf(resp.String())
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to unmarshall response: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}

	token := result["access_token"].(string)
	client.Token = token
	gitlab.Client = *client

	return nil
}

func (gitlab *GitLab) CheckProjectExist(projectPath string) (bool, error) {
	resp, err := gitlab.Client.R().
		SetAuthToken(gitlab.Client.Token).
		Get(gitlab.Client.HostURL + "/api/v4/projects/" + projectPath + "?simple=true")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to check project: %v", err)
		log.Println(errorMsg)
		return true, errors.New(errorMsg)
	}
	return resp.StatusCode() != 404, nil
}

func (gitlab *GitLab) GetGroupIdByName(groupName string) (string, error) {
	resp, err := gitlab.Client.R().
		SetAuthToken(gitlab.Client.Token).
		Get(gitlab.Client.HostURL + "/api/v4/groups/" + url.PathEscape(groupName) + "?simple=true")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable get repository group id: %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}
	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to unmarshall response: %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}
	id := result["id"].(float64)

	return strconv.FormatFloat(id, 'f', -1, 64), nil
}

func (gitlab *GitLab) CreateProject(vcsProjectName string, vcsGroupId string) error {
	resp, err := gitlab.Client.R().
		SetAuthToken(gitlab.Client.Token).
		Post(gitlab.Client.HostURL + "/api/v4/projects?name=" + vcsProjectName + "&namespace_id=" + vcsGroupId)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to create project in GitLab: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	if resp.IsError() {
		errorMsg := fmt.Sprintf(resp.String())
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	return nil
}
