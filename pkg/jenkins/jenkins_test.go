package jenkins

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestExampleJenkins_TriggerBuildJob(t *testing.T) {
	url := "https://jenkins-qa-edp-cicd.delivery.aws.main.edp.projects.epam.com/"
	appName := "test-1"
	username := "admin"
	token := ""

	jenkinsInstance, err := Init(url, username, token)
	if err != nil {
		log.Print(err)
	}

	err = jenkinsInstance.TriggerBuildJob(appName)
	if err != nil {
		log.Print(err)
	}
}

func TestExampleJenkins_TriggerReleaseJob(t *testing.T) {
	url := "https://jenkins-mr-1617-2-edp-cicd.delivery.aws.main.edp.projects.epam.com/"
	appName := "test-1"
	username := "admin"
	token := ""

	jenkinsInstance, err := Init(url, username, token)
	if err != nil {
		log.Print(err)
	}

	err = jenkinsInstance.TriggerReleaseJob("test-1-release-1.0.0", "fe144d35dbbab8688b17444e2705ec139e73c3c0", appName)
	if err != nil {
		log.Print(err)
	}
}

func TestExampleJenkins_GetJobStatus(t *testing.T) {
	url := "https://jenkins-mr-1617-5-edp-cicd.delivery.aws.main.edp.projects.epam.com"
	JobName := "Create-release-test-npm"
	username := "admin"
	token := ""

	jenkinsInstance, err := Init(url, username, token)
	if err != nil {
		log.Print(err)
	}

	status, err := jenkinsInstance.GetJobStatus(JobName, time.Second, 30)
	if err != nil {
		log.Print(err)
	}
	fmt.Println(status)
}
