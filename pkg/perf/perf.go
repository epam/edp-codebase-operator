package perf

import (
	"errors"
	"fmt"
	ClientSet "github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"gopkg.in/resty.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"reflect"
	"strconv"
)

type Client struct {
	client resty.Client
}

const (
	UrlSettingsKey       = "url"
	JenkinsDsSettingsKey = "jenkins_ds_id"
	SonarDsSettingsKey   = "sonar_ds_id"
	GerritDsSettingsKey  = "gerrit_ds_id"
	GitlabDsSettingsKey  = "gitlab_ds_id"
)

type DataSource struct {
	Id     int                    `json:"id"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type GerritPerfConfig struct {
	ProjectName string   `json:"projectName"`
	Branches    []string `json:"branches"`
}

func NewRestClient(url string, user string, pass string) (*Client, error) {
	token, err := getToken(url, user, pass)
	if err != nil {
		return nil, err
	}
	cl := resty.New().
		SetHostURL(url).
		SetAuthToken(token)
	return &Client{
		client: *cl,
	}, err
}

func (perf Client) AddJobsToJenkinsDS(dsId int, jobs []string) error {
	ds, err := perf.getDatasource(dsId)
	if err != nil {
		return err
	}
	addToDsConfig(*ds, "jobNames", jobs)
	return perf.updateDatasource(*ds)
}

func (perf Client) AddProjectsToSonarDS(dsId int, projects []string) error {
	ds, err := perf.getDatasource(dsId)
	if err != nil {
		return err
	}
	addToDsConfig(*ds, "projectKeys", projects)
	return perf.updateDatasource(*ds)
}

func (perf Client) AddProjectsToGerritDS(dsId int, projects []GerritPerfConfig) error {
	ds, err := perf.getDatasource(dsId)
	if err != nil {
		return err
	}
	addToDsConfig(*ds, "projectConfigs", projects)
	return perf.updateDatasource(*ds)
}

func (perf Client) AddRepositoriesToGitlabDS(dsId int, repos map[string]string) error {
	ds, err := perf.getDatasource(dsId)
	if err != nil {
		return err
	}
	addToDsConfigMap(*ds, "repositories", repos)
	return perf.updateDatasource(*ds)
}

func getToken(url string, user string, pass string) (string, error) {
	resp, err := resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"username": user,
			"password": pass,
		}).Post(url + "/api/v2/sso/token")

	if err != nil || resp.IsError() {
		errorMsg := fmt.Sprintf("Failed to get perf token. %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}

	return resp.String(), nil
}

func (perf Client) getDatasource(id int) (*DataSource, error) {
	log.Printf("Start retrieving perf datasource with id %v by url %s", id, perf.client.HostURL)
	var ds DataSource
	resp, err := perf.client.R().
		SetHeader("Content-Type", "application/json").
		SetResult(&ds).
		SetPathParams(map[string]string{
			"id": strconv.Itoa(id),
		}).
		Get("/api/v2/datasources/{id}")
	if err != nil || resp.IsError() {
		errorMsg := fmt.Sprintf("Failed to get datasource. %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	return &ds, nil
}

func (perf Client) updateDatasource(ds DataSource) error {
	log.Printf("Start updating perf datasource with id %v by url %s", ds.Id, perf.client.HostURL)
	resp, err := perf.client.R().
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"id": strconv.Itoa(ds.Id),
		}).
		SetBody(ds).
		Put("/api/v2/datasources/{id}")
	if err != nil || resp.IsError() {
		errorMsg := fmt.Sprintf("Failed to update datasource. %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	return nil
}

func addToDsConfig(ds DataSource, configKey string, configs interface{}) {
	slice := reflect.ValueOf(ds.Config[configKey])
	sliceNewConfig := reflect.ValueOf(configs)
	arr := make([]interface{}, slice.Len()+sliceNewConfig.Len())

	for i := 0; i < slice.Len(); i++ {
		arr[i] = slice.Index(i).Interface()
	}

	for i := 0; i < sliceNewConfig.Len(); i++ {
		arr[i+slice.Len()] = sliceNewConfig.Index(i).Interface()
	}
	ds.Config[configKey] = arr
}

func addToDsConfigMap(ds DataSource, configKey string, configs map[string]string) {
	switch curConfig := ds.Config[configKey].(type) {
	case map[string]interface{}:
		for k, v := range configs {
			curConfig[k] = v
		}
	}
}

func GetPerfSettings(clientSet ClientSet.ClientSet, namespace string) map[string]string {
	perfSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("perf-settings", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return perfSettings.Data
}

func GetPerfCredentials(clientSet ClientSet.ClientSet, namespace string) map[string][]byte {
	secret, err := clientSet.CoreClient.Secrets(namespace).Get("perf-user", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return secret.Data
}
