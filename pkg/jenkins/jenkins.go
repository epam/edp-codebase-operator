package jenkins

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	jenkins gojenkins.Jenkins
}

func Init(url string, username string, token string) (*Client, error) {
	jenkins := gojenkins.CreateJenkins(&http.Client{}, url, username, token)

	_, err := jenkins.Init()
	if err != nil {
		return nil, err
	}

	return &Client{
		jenkins: *jenkins,
	}, nil
}

func (client Client) TriggerJobProvisioning(appName string, buildTool string) error {
	_, err := client.jenkins.BuildJob("Job-provisioning", map[string]string{
		"PARAM":      "true",
		"NAME":       appName,
		"TYPE":       "app",
		"BUILD_TOOL": strings.ToLower(buildTool),
	})
	return err
}

func (client Client) GetJob(name string, delay time.Duration, retryCount int) bool {
	for i := 0; i < retryCount; i++ {
		_, err := client.jenkins.GetJob(name)
		if err == nil {
			return true
		}
		log.Printf("Job %v is currently doesn't exist. Sleeping for %v. %v attempts lasts",
			name, delay, retryCount-i)
		time.Sleep(delay)
	}
	return false
}

func (client Client) TriggerBuildJob(appName string) error {
	jobName := fmt.Sprintf("MASTER-Build-%v", appName)
	log.Printf("Trying to trigger jenkins job %v", jobName)

	if client.GetJob(jobName, time.Second, 60) {
		_, err := client.jenkins.BuildJob(jobName, map[string]string{
			"GERRIT_PROJECT_NAME": appName,
			"BRANCH":              "master",
		})
		return err
	}
	return errors.New(fmt.Sprintf("Couldn't trigger %v job", jobName))
}
