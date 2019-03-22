package jenkins

import (
	"log"
	"testing"
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