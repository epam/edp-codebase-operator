package jenkins

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
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
