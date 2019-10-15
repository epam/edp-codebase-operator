package jenkins

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
	"log"
	"net/http"
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

func (client Client) TriggerJobProvisioning(jobName string, parameters map[string]string, delay time.Duration, retryCount int) error {
	for i := 0; i < retryCount; i++ {
		buildNumber, err := client.jenkins.BuildJob("job-provisions/job/"+jobName, parameters)
		if buildNumber != 0 || err != nil {
			return err
		}
		time.Sleep(delay)
	}
	return errors.Errorf("Job provisioning for %v has not been finished after specified delay", parameters["NAME"])
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
	jobName := fmt.Sprintf("%v/job/MASTER-Build-%v", appName, appName)
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

func (client Client) TriggerReleaseJob(branchName string, fromCommit string, appName string) error {
	jobName := fmt.Sprintf("%v/job/Create-release-%v", appName, appName)
	log.Printf("Trying to trigger jenkins job %v", jobName)

	if client.GetJob(jobName, time.Second, 60) {
		_, err := client.jenkins.BuildJob(jobName, map[string]string{
			"RELEASE_NAME": branchName,
			"COMMIT_ID":    fromCommit,
		})
		return err
	}
	return errors.New(fmt.Sprintf("Couldn't trigger %v job", jobName))
}

func (client Client) GetJobStatus(name string, delay time.Duration, retryCount int) (string, error) {
	time.Sleep(delay)
	for i := 0; i < retryCount; i++ {
		isQueued, err := client.IsJobQueued(name)
		isRunning, err := client.IsJobRunning(name)
		if err != nil {
			job, err := client.jenkins.GetJob(name)
			if job.Raw.Color == "notbuilt" {
				log.Printf("Job %v didn't start yet. Sleeping for %v. %v attempts lasts",
					name, delay, retryCount-i)
				time.Sleep(delay)
				continue
			}

			if err != nil {
				return "", err
			}
		}
		if *isRunning || *isQueued {
			log.Printf("Job %v is running. Sleeping for %v. %v attempts lasts",
				name, delay, retryCount-i)
			time.Sleep(delay)
		} else {
			job, err := client.jenkins.GetJob(name)
			if err != nil {
				return "", err
			}

			return job.Raw.Color, nil
		}
	}

	return "", errors.Errorf("Job %v has not been finished after specified delay", name)
}

func (client Client) IsJobQueued(name string) (*bool, error) {
	job, err := client.jenkins.GetJob(name)
	if err != nil {
		return nil, err
	}

	isQueued, err := job.IsQueued()
	if err != nil {
		return nil, err
	}

	return &isQueued, nil
}

func (client Client) IsJobRunning(name string) (*bool, error) {
	job, err := client.jenkins.GetJob(name)
	if err != nil {
		return nil, err
	}

	isRunning, err := job.IsRunning()
	if err != nil {
		return nil, err
	}

	return &isRunning, nil
}
