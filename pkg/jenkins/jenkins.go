package jenkins

import (
	"context"
	"fmt"
	"github.com/bndr/gojenkins"
	ClientSet "github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsOperatorSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
	"time"
)

var log = logf.Log.WithName("jenkins-client")

type JenkinsClient struct {
	Jenkins *gojenkins.Jenkins
}

func Init(url string, username string, token string) (*JenkinsClient, error) {
	log.Info("initializing new Jenkins client", "url", url, "username", username)
	jenkins, err := gojenkins.CreateJenkins(&http.Client{}, url, username, token).Init()
	if err != nil {
		return nil, err
	}
	return &JenkinsClient{
		Jenkins: jenkins,
	}, nil
}

func (c JenkinsClient) TriggerJobProvisioning(jobName string, parameters map[string]string) (*int64, error) {
	jn := fmt.Sprintf("job-provisions/job/%v", jobName)
	log.Info("start triggering job provision", "name", jn, "codebase name", parameters["NAME"])
	qn, err := c.Jenkins.BuildJob(jn, parameters)
	if qn != 0 || err != nil {
		log.Info("end triggering job provision", "name", jn, "codebase name", parameters["NAME"])
		return c.getBuildNumber(qn)
	}
	return nil, errors.Errorf("couldn't finish triggering job provision for %v codebase", parameters["NAME"])
}

func (c JenkinsClient) getBuildNumber(queueNumber int64) (*int64, error) {
	log.Info("start getting build number", "queueNumber", queueNumber)
	for i := 0; i < 3; i++ {
		t, err := c.Jenkins.GetQueueItem(queueNumber)
		if err != nil {
			return nil, err
		}
		n := t.Raw.Executable.Number
		if n != 0 {
			log.Info("end getting build number", "number", n)
			return &n, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("couldn't get build number by queue number %v", queueNumber)
}

func (c JenkinsClient) GetJob(name string, delay time.Duration, retryCount int) bool {
	for i := 0; i < retryCount; i++ {
		_, err := c.Jenkins.GetJob(name)
		if err == nil {
			return true
		}
		log.Info("Job is currently doesn't exist", "name", name, "delay", delay, "attempts lasts", retryCount-i)
		time.Sleep(delay)
	}
	return false
}

func (c JenkinsClient) TriggerBuildJob(appName string) error {
	jobName := fmt.Sprintf("%v/job/MASTER-Build-%v", appName, appName)
	log.Info("Trying to trigger jenkins job", "name", jobName)

	if c.GetJob(jobName, time.Second, 60) {
		_, err := c.Jenkins.BuildJob(jobName, map[string]string{
			"GERRIT_PROJECT_NAME": appName,
			"BRANCH":              "master",
		})
		return err
	}
	return errors.New(fmt.Sprintf("Couldn't trigger %v job", jobName))
}

func (c JenkinsClient) TriggerReleaseJob(branchName string, fromCommit string, appName string) error {
	jobName := fmt.Sprintf("%v/job/Create-release-%v", appName, appName)
	log.Info("Trying to trigger jenkins job", "name", jobName)

	if c.GetJob(jobName, time.Second, 60) {
		_, err := c.Jenkins.BuildJob(jobName, map[string]string{
			"RELEASE_NAME": branchName,
			"COMMIT_ID":    fromCommit,
		})
		return err
	}
	return errors.New(fmt.Sprintf("Couldn't trigger %v job", jobName))
}

func (c JenkinsClient) GetJobStatus(name string, delay time.Duration, retryCount int) (string, error) {
	time.Sleep(delay)
	for i := 0; i < retryCount; i++ {
		isQueued, err := c.IsJobQueued(name)
		isRunning, err := c.IsJobRunning(name)
		if err != nil {
			job, err := c.Jenkins.GetJob(name)
			if job.Raw.Color == "notbuilt" {
				log.Info("Job didn't start yet", "name", name, "delay", delay, "attempts lasts", retryCount-i)
				time.Sleep(delay)
				continue
			}

			if err != nil {
				return "", err
			}
		}
		if *isRunning || *isQueued {
			log.Info("Job is running", "name", name, "delay", delay, "attempts lasts", retryCount-i)
			time.Sleep(delay)
		} else {
			job, err := c.Jenkins.GetJob(name)
			if err != nil {
				return "", err
			}

			return job.Raw.Color, nil
		}
	}

	return "", errors.Errorf("Job %v has not been finished after specified delay", name)
}

func (c JenkinsClient) IsJobQueued(name string) (*bool, error) {
	job, err := c.Jenkins.GetJob(name)
	if err != nil {
		return nil, err
	}

	isQueued, err := job.IsQueued()
	if err != nil {
		return nil, err
	}

	return &isQueued, nil
}

func (c JenkinsClient) IsJobRunning(name string) (*bool, error) {
	job, err := c.Jenkins.GetJob(name)
	if err != nil {
		return nil, err
	}

	isRunning, err := job.IsRunning()
	if err != nil {
		return nil, err
	}

	return &isRunning, nil
}

func (c JenkinsClient) IsBuildSuccessful(jobName string, buildNumber int64) (bool, error) {
	log.Info("start checking build", "job name", jobName, "build number", buildNumber)
	jp := fmt.Sprintf("job-provisions/job/%s", jobName)
	job, err := c.Jenkins.GetJob(jp)
	if err != nil {
		return false, errors.Wrapf(err, "could't get job %v", jp)
	}

	b, err := getBuild(jp, job, buildNumber)
	if err != nil {
		if err.Error() == "404" {
			log.Info("couldn't find build", "build number", buildNumber)
			return false, nil
		}
		return false, err
	}
	return b.GetResult() == "SUCCESS", nil
}

func getBuild(jp string, job *gojenkins.Job, id int64) (*gojenkins.Build, error) {
	endpoint := "/job/" + jp
	build := gojenkins.Build{Jenkins: job.Jenkins, Job: job, Raw: new(gojenkins.BuildResponse), Depth: 1, Base: endpoint + "/" + strconv.FormatInt(id, 10)}
	status, err := build.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &build, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}

func GetJenkins(c client.Client, namespace string) (*jenkinsApi.Jenkins, error) {
	options := client.ListOptions{Namespace: namespace}
	jenkinsList := &jenkinsApi.JenkinsList{}

	err := c.List(context.TODO(), &options, jenkinsList)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Jenkins CRs in namespace %v", namespace)
	}

	if len(jenkinsList.Items) == 0 {
		return nil, fmt.Errorf("jenkins installation is not found in namespace %v", namespace)
	}

	return &jenkinsList.Items[0], nil
}

func GetJenkinsCreds(jenkins jenkinsApi.Jenkins, clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	annotationKey := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, jenkinsOperatorSpec.JenkinsTokenAnnotationSuffix)
	jenkinsTokenSecretName := jenkins.Annotations[annotationKey]
	jenkinsTokenSecret, err := clientSet.CoreClient.Secrets(namespace).Get(jenkinsTokenSecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Secret %v in not found", jenkinsTokenSecretName)
		}
		return "", "", errors.Wrapf(err, "Getting secret %v failed", jenkinsTokenSecretName)
	}
	return string(jenkinsTokenSecret.Data["password"]), string(jenkinsTokenSecret.Data["username"]), nil
}

func GetJenkinsUrl(jenkins jenkinsApi.Jenkins, namespace string) string {
	log.Info("creating Jenkins url")
	key := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, "externalUrl")
	url := jenkins.Annotations[key]
	if len(url) == 0 {
		url := fmt.Sprintf("http://jenkins.%s:8080", namespace)
		log.Info("annotation doesn't contain Jenkins url. creating from template", "url", url)
		return url
	}
	log.Info("Jenkins url is taken from annotation", "url", url)
	return url
}
