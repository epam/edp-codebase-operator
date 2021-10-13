package jenkins

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestJenkinsClient_TriggerReleaseJob_JobNotFound(t *testing.T) {
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(200, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins:                  jenkins,
		triggerReleaseRetryCount: 1,
	}

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewStringResponder(404, ""))

	err = jc.TriggerReleaseJob("codebase", map[string]string{"foo": "bar"})
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "unable to get job") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestJenkinsClient_TriggerReleaseJob_UnableToBuild(t *testing.T) {
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(200, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins:                  jenkins,
		triggerReleaseRetryCount: 1,
	}

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewStringResponder(404, ""))

	jrsp := gojenkins.JobResponse{}

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "j-url/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	httpmock.RegisterResponder("POST", "j-url/job/codebase/job/Create-release-codebase/build",
		httpmock.NewStringResponder(500, ""))

	err = jc.TriggerReleaseJob("codebase", map[string]string{"foo": "bar"})
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "Couldn't trigger") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestJenkinsClient_TriggerReleaseJob(t *testing.T) {
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(200, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins: jenkins,
	}

	jrsp := gojenkins.JobResponse{}

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "j-url/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "j-url/job/codebase/job/Create-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	if err := jc.TriggerReleaseJob("codebase", map[string]string{"foo": "bar"}); err != nil {
		t.Fatal(err)
	}
}

func TestJenkinsClient_IsJobQueued_True(t *testing.T) {
	// TODO: Move shared code to SetupTest() to meet DRY concept
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(200, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins: jenkins,
	}

	jrsp := gojenkins.JobResponse{InQueue: true}

	httpmock.RegisterResponder("GET", "j-url/job/queued-job/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	isQueued, err := jc.IsJobQueued("queued-job")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, *isQueued)
}

func TestJenkinsClient_IsJobQueued_False(t *testing.T) {
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(200, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins: jenkins,
	}

	jrsp := gojenkins.JobResponse{InQueue: false}

	httpmock.RegisterResponder("GET", "j-url/job/non-queued-job/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	isQueued, err := jc.IsJobQueued("non-queued-job")
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, *isQueued)
}

func TestJenkinsClient_IsJobQueued_JobNotFound(t *testing.T) {
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(200, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins: jenkins,
	}

	httpmock.RegisterResponder("GET", "j-url/job/not-found-job/api/json",
		httpmock.NewStringResponder(404, ""))

	isQueued, err := jc.IsJobQueued("not-found-job")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Nil(t, isQueued)
}
