package jenkins

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bndr/gojenkins"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsOperatorSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("jenkins-client")

type JenkinsClient struct {
	Jenkins                  *gojenkins.Jenkins
	triggerReleaseRetryCount int
}

type JobNotFoundError string

func (j JobNotFoundError) Error() string {
	return string(j)
}

func Init(url, username, token string) (*JenkinsClient, error) {
	const defaultRetryCount = 60

	log.Info("initializing new Jenkins client", "url", url, "username", username)

	jenkins, err := gojenkins.CreateJenkins(http.DefaultClient, url, username, token).Init()
	if err != nil {
		return nil, fmt.Errorf("failed to create Jenkins client: %w", err)
	}

	return &JenkinsClient{
		Jenkins:                  jenkins,
		triggerReleaseRetryCount: defaultRetryCount,
	}, nil
}

func (c JenkinsClient) GetJob(name string, delay time.Duration, retryCount int) (*gojenkins.Job, error) {
	var resultErr error

	for i := 0; i < retryCount; i++ {
		job, err := c.Jenkins.GetJob(name)
		if err == nil {
			return job, nil
		}

		resultErr = err

		log.Info("Job is currently doesn't exist", "name", name, "delay", delay, "attempts lasts", retryCount-i)
		time.Sleep(delay)
	}

	return nil, resultErr
}

func (c JenkinsClient) TriggerDeletionJob(branchName, appName string) error {
	jobName := fmt.Sprintf("%v/job/Delete-release-%v", appName, appName)
	log.Info("Trying to trigger Deletion jenkins job", "name", jobName)

	job, err := c.GetJob(jobName, time.Second, 1)
	if err != nil {
		return JobNotFoundError(err.Error())
	}

	lastBuild, err := job.GetLastBuild()
	if err != nil && err.Error() != "404" {
		return fmt.Errorf("failed to fetch Jenkins Job build: %w", err)
	}

	if (lastBuild != nil && lastBuild.IsRunning()) || job.Raw.InQueue {
		return nil
	}

	_, err = c.Jenkins.BuildJob(jobName, map[string]string{
		"RELEASE_NAME": branchName,
	})
	if err != nil {
		return fmt.Errorf("failed to build job: %w", err)
	}

	return nil
}

func (c JenkinsClient) TriggerReleaseJob(appName string, params map[string]string) error {
	jobName := fmt.Sprintf("%v/job/Create-release-%v", appName, appName)
	log.Info("Trying to trigger Release jenkins job", "jobName", jobName)

	if _, err := c.GetJob(jobName, time.Second, c.triggerReleaseRetryCount); err != nil {
		return fmt.Errorf("failed to get job %s: %w", jobName, err)
	}

	if _, err := c.Jenkins.BuildJob(jobName, params); err != nil {
		return fmt.Errorf("failed to trigger %v job: %w", jobName, err)
	}

	return nil
}

func (c JenkinsClient) GetJobStatus(name string, delay time.Duration, retryCount int) (string, error) {
	time.Sleep(delay)

	for i := 0; i < retryCount; i++ {
		isQueued, qErr := c.IsJobQueued(name)
		isRunning, rErr := c.IsJobRunning(name)

		if qErr != nil || rErr != nil {
			job, err := c.Jenkins.GetJob(name)
			if err != nil {
				return "", fmt.Errorf("job not found: %w", err)
			}

			if job.Raw.Color == "notbuilt" {
				log.Info("Job didn't start yet", "jobName", name, "delay", delay, "attempts lasts", retryCount-i)
				time.Sleep(delay)

				continue
			}
		}

		if !(isRunning != nil && *isRunning) && !(isQueued != nil && *isQueued) {
			job, err := c.Jenkins.GetJob(name)
			if err != nil {
				return "", fmt.Errorf("failed to get Jenkins Job: %w", err)
			}

			return job.Raw.Color, nil
		}

		log.Info("Job is running", "jobName", name, "delay", delay, "attempts lasts", retryCount-i)

		time.Sleep(delay)
	}

	return "", fmt.Errorf("failed to finish job after specified delay - job %v", name)
}

func (c JenkinsClient) IsJobQueued(name string) (*bool, error) {
	job, err := c.Jenkins.GetJob(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jenkins Job: %w", err)
	}

	isQueued, err := job.IsQueued()
	if err != nil {
		return nil, fmt.Errorf("failed to check if Jenkins Job is queued: %w", err)
	}

	return &isQueued, nil
}

func (c JenkinsClient) IsJobRunning(name string) (*bool, error) {
	job, err := c.Jenkins.GetJob(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jenkins Job: %w", err)
	}

	isRunning, err := job.IsRunning()
	if err != nil {
		return nil, fmt.Errorf("failed to check if Jenkins Job is bulding: %w", err)
	}

	return &isRunning, nil
}

func GetJenkins(c client.Client, namespace string) (*jenkinsApi.Jenkins, error) {
	options := client.ListOptions{Namespace: namespace}
	jenkinsList := &jenkinsApi.JenkinsList{}

	err := c.List(context.TODO(), jenkinsList, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jenkins CRs in namespace %v: %w", namespace, err)
	}

	if len(jenkinsList.Items) == 0 {
		return nil, fmt.Errorf("jenkins installation is not found in namespace %v", namespace)
	}

	return &jenkinsList.Items[0], nil
}

func GetJenkinsCreds(c client.Client, jenkins *jenkinsApi.Jenkins, namespace string) (userName, password string, err error) {
	annotationKey := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, jenkinsOperatorSpec.JenkinsTokenAnnotationSuffix)
	jenkinsTokenSecretName := jenkins.Annotations[annotationKey]

	jenkinsTokenSecret, err := util.GetSecret(c, jenkinsTokenSecretName, namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "", "", fmt.Errorf("Secret %v in not found: %w", jenkinsTokenSecretName, err)
		}

		return "", "", fmt.Errorf("Getting secret %v failed: %w", jenkinsTokenSecretName, err)
	}

	userName = string(jenkinsTokenSecret.Data["username"])
	password = string(jenkinsTokenSecret.Data["password"])

	return
}

func GetJenkinsUrl(jenkins *jenkinsApi.Jenkins, namespace string) string {
	log.Info("creating Jenkins url")

	key := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, "externalUrl")
	url := jenkins.Annotations[key]
	basePath := ""

	if len(jenkins.Spec.BasePath) > 0 {
		basePath = fmt.Sprintf("/%v", jenkins.Spec.BasePath)
	}

	if url == "" {
		return fmt.Sprintf("http://jenkins.%s:8080%v", namespace, basePath)
	}

	return url
}
