package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

type BitBucket struct {
	Client resty.Client
}
type project struct {
	Links links `json:"links"`
}

type links struct {
	Clone []link `json:"clone"`
}

type link struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

type repository struct {
	ID int `json:"id"`
}

const retryCount = 3

func (bitBucket *BitBucket) Init(url, username, password string) error {
	client := resty.New()
	client.SetRetryCount(retryCount)
	client.HostURL = url
	client.AddRetryCondition(
		func(response *resty.Response) (bool, error) {
			return response.IsError(), nil
		},
	)

	client.SetBasicAuth(username, password)
	bitBucket.Client = *client

	return nil
}

func (bitBucket *BitBucket) CheckProjectExist(groupPath, projectName string) (*bool, error) {
	_, exist, err := bitBucket.getProject(groupPath, projectName)
	if err != nil {
		return nil, err
	}

	return exist, nil
}

func (bitBucket *BitBucket) getProject(groupPath, projectName string) (*project, *bool, error) {
	var result project

	resp, err := bitBucket.Client.R().
		SetResult(&result).
		SetPathParams(map[string]string{
			"groupPath":   groupPath,
			"projectName": projectName,
		}).
		Get("/rest/api/1.0/projects/{groupPath}/repos/{projectName}")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to read project: %v", err)
		log.Println(errorMsg)

		return nil, nil, errors.New(errorMsg)
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		errorMsg := "unauthorized"
		log.Println(errorMsg)

		return nil, nil, errors.New(errorMsg)
	}

	var exist bool
	if resp.StatusCode() == http.StatusNotFound {
		exist = false
		return nil, &exist, nil
	}

	exist = resp.StatusCode() == http.StatusOK

	return &result, &exist, nil
}

func (bitBucket *BitBucket) CreateProject(groupPath, projectName string) (string, error) {
	var result repository

	resp, err := bitBucket.Client.R().
		SetResult(&result).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"name":       projectName,
			"is_private": false,
		}).
		SetPathParams(map[string]string{
			"groupPath": groupPath,
		}).
		Post("/rest/api/1.0/projects/{groupPath}/repos")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to create project in Bitbucket: %v", err)
		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	if resp.IsError() {
		errorMsg := resp.String()
		log.Println(errorMsg)

		return "", errors.New(errorMsg)
	}

	return strconv.Itoa(result.ID), nil
}

func (bitBucket *BitBucket) GetRepositorySshUrl(groupPath, projectName string) (string, error) {
	pr, exist, err := bitBucket.getProject(groupPath, projectName)
	if err != nil {
		return "", err
	}

	if !*exist {
		return "", errors.Errorf("project %v, does not exist in group %v", projectName, groupPath)
	}

	sshLink, err := getSshLink(*pr)
	if err != nil {
		return "", err
	}

	return *sshLink, nil
}

func getSshLink(pr project) (*string, error) {
	links := pr.Links.Clone
	for _, el := range links {
		if el.Name == "ssh" {
			return &el.Href, nil
		}
	}

	return nil, errors.New("ssh link has not be found")
}

func (bitBucket *BitBucket) DeleteProject(groupPath, projectName string) error {
	resp, err := bitBucket.Client.R().
		SetPathParams(map[string]string{
			"groupPath":   groupPath,
			"projectName": projectName,
		}).
		Delete("/rest/api/1.0/projects/{groupPath}/repos/{projectName}")
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to delete project in Bitbucket: %v", err)
		log.Println(errorMsg)

		return errors.New(errorMsg)
	}

	if resp.IsError() {
		log.Println(resp.Status())

		return errors.New(resp.Status())
	}

	return nil
}
