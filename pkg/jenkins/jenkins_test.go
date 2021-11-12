package jenkins

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	fakeName      = "fake-name"
	fakeNamespace = "fake-namespace"
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

func TestJenkinsClient_IsJobRunning_True(t *testing.T) {
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

	httpmock.RegisterResponder("GET", "j-url/job/running-job/api/json",
		httpmock.NewStringResponder(200, ""))

	brsp := gojenkins.BuildResponse{Building: true}

	httpmock.RegisterResponder("GET", "j-url/job/running-job/0/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))

	isRunning, err := jc.IsJobRunning("running-job")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, *isRunning)
}

func TestJenkinsClient_IsJobRunning_False(t *testing.T) {
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

	httpmock.RegisterResponder("GET", "j-url/job/not-running-job/api/json",
		httpmock.NewStringResponder(200, ""))

	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "j-url/job/not-running-job/0/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))

	isRunning, err := jc.IsJobRunning("not-running-job")
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, *isRunning)
}

func TestJenkinsClient_IsJobRunning_JobNotFound(t *testing.T) {
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

	isRunning, err := jc.IsJobRunning("not-found-job")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Nil(t, isRunning)
}

func TestJenkinsClient_IsJobRunning_JobFoundButError(t *testing.T) {
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

	httpmock.RegisterResponder("GET", "j-url/job/found-job/api/json",
		httpmock.NewStringResponder(200, ""))

	isRunning, err := jc.IsJobRunning("found-job")

	if !strings.Contains(err.Error(), "no responder found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Error(t, err)
	assert.Nil(t, isRunning)
}

func TestGetJenkinsUrl_UrlOnly(t *testing.T) {

	jspec := v1alpha1.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
			Annotations: map[string]string{
				"edp.epam.com/externalUrl": "external-jenkins",
			},
		},
		Spec: v1alpha1.JenkinsSpec{},
	}

	jurl := GetJenkinsUrl(jspec, "namespace")

	assert.Equal(t, "external-jenkins", jurl)
}

func TestGetJenkinsUrl_BasepathOnly(t *testing.T) {

	jspec := v1alpha1.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.JenkinsSpec{
			BasePath: "basepath",
		},
	}

	jurl := GetJenkinsUrl(jspec, "namespace")

	assert.Equal(t, "http://jenkins.namespace:8080/basepath", jurl)
}

func TestGetJenkinsUrl_WithUrlAndBasepath(t *testing.T) {

	jspec := v1alpha1.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
			Annotations: map[string]string{
				"edp.epam.com/externalUrl": "external-jenkins",
			},
		},
		Spec: v1alpha1.JenkinsSpec{
			BasePath: "basepath",
		},
	}

	jurl := GetJenkinsUrl(jspec, "namespace")

	assert.Equal(t, "external-jenkins", jurl)
}

func TestGetJenkinsUrl_NoUrlNoBasepath(t *testing.T) {

	jspec := v1alpha1.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	jurl := GetJenkinsUrl(jspec, "namespace")

	assert.Equal(t, "http://jenkins.namespace:8080", jurl)
}

func TestGetJenkinsCreds_SecretExists(t *testing.T) {

	s := &coreV1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fake-admin-token",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("j-user"),
			"password": []byte("j-token"),
		},
	}

	jspec := &v1alpha1.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
			Annotations: map[string]string{
				"edp.epam.com/admin-token": "fake-admin-token",
			},
		},
		Spec: v1alpha1.JenkinsSpec{},
	}

	objs := []runtime.Object{
		s, jspec,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, jspec)
	client := fake.NewFakeClient(objs...)

	jt, ju, err := GetJenkinsCreds(client, *jspec, fakeNamespace)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "j-token", jt)
	assert.Equal(t, "j-user", ju)
}

func TestGetJenkinsCreds_NoSecretExists(t *testing.T) {

	jspec := &v1alpha1.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
			Annotations: map[string]string{
				"edp.epam.com/admin-token": "non-existing-secret",
			},
		},
		Spec: v1alpha1.JenkinsSpec{},
	}

	objs := []runtime.Object{
		jspec,
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, jspec)
	client := fake.NewFakeClient(objs...)

	jt, ju, err := GetJenkinsCreds(client, *jspec, fakeNamespace)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "Secret non-existing-secret in not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.Empty(t, jt)
	assert.Empty(t, ju)
}

func TestGetJenkins_ShouldFailWhenNotFound(t *testing.T) {
	jl := &v1alpha1.JenkinsList{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, jl)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(jl).Build()

	j, err := GetJenkins(fakeCl, fakeNamespace)
	assert.Nil(t, j)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "jenkins installation is not found in namespace fake-namespace") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetJenkins_ShouldPass(t *testing.T) {

	j := &v1alpha1.Jenkins{}
	jl := &v1alpha1.JenkinsList{
		Items: []v1alpha1.Jenkins{
			{
				ObjectMeta: v1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, j, jl)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j, jl).Build()

	j, err := GetJenkins(fakeCl, fakeNamespace)
	assert.NoError(t, err)
	assert.Equal(t, j.Name, fakeName)
}

func TestJenkinsClient_TriggerDeletionJob_JobNotFound(t *testing.T) {
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

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewStringResponder(404, ""))

	err = jc.TriggerDeletionJob("master", "codebase")
	if err == nil {
		t.Fatal("no error returned")
	}

	if errors.Cause(err) != JobNotFoundError(err.Error()) {
		t.Fatal("wrong error returned")
	}
}

func TestJenkinsClient_TriggerDeletionJob_LastBuildNotFoundIssue(t *testing.T) {
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

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewStringResponder(200, ""))

	err = jc.TriggerDeletionJob("master", "codebase")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "no responder found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestJenkinsClient_TriggerDeletionJob_ShouldPass(t *testing.T) {
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		LastBuild: gojenkins.JobBuild{
			Number: 10,
			URL:    "some",
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Delete-release-codebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))

	httpmock.RegisterResponder("GET", "j-url/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "j-url/job/codebase/job/Delete-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	if err := jc.TriggerDeletionJob("master", "codebase"); err != nil {
		t.Fatal(err)
	}
}

func TestJenkinsClient_TriggerDeletionJob_ShouldFailOnJobBuildFailure(t *testing.T) {
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		LastBuild: gojenkins.JobBuild{
			Number: 10,
			URL:    "some",
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	httpmock.RegisterResponder("GET", "j-url/job/codebase/job/Delete-release-codebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))

	httpmock.RegisterResponder("GET", "j-url/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	httpmock.RegisterResponder("POST", "j-url/job/codebase/job/Delete-release-codebase/build",
		httpmock.NewStringResponder(500, ""))

	err = jc.TriggerDeletionJob("master", "codebase")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "unable to build job") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestJenkinsClient_GetJobStatus_ShouldPass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "Green",
	}
	httpmock.RegisterResponder("GET", "j-url/job/job-name/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	brsp := gojenkins.BuildResponse{Building: false}
	httpmock.RegisterResponder("GET", "j-url/job/job-name/0/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))

	js, err := jc.GetJobStatus("job-name", 1*time.Millisecond, 1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, js, "Green")
}

func TestJenkinsClient_GetJobStatus_ShouldFailWithJobNotFound(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()
	httpClient := http.Client{}
	httpmock.ActivateNonDefault(&httpClient)
	httpmock.RegisterResponder("GET", "j-url/api/json", httpmock.NewStringResponder(404, ""))
	jenkins, err := gojenkins.CreateJenkins(&httpClient, "j-url", "j-username", "j-token").Init()
	if err != nil {
		t.Fatal(err)
	}

	jc := JenkinsClient{
		Jenkins: jenkins,
	}

	js, err := jc.GetJobStatus("job-name", 1*time.Millisecond, 1)
	assert.Error(t, err)
	assert.Equal(t, js, "")
	assert.Contains(t, err.Error(), "job not found")
}

func TestJenkinsClient_GetJobStatus_ShouldFailOnTimeout(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "notbuilt",
	}
	httpmock.RegisterResponder("GET", "j-url/job/job-name/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	brsp := gojenkins.BuildResponse{Building: true}
	httpmock.RegisterResponder("GET", "j-url/job/job-name/0/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))

	js, err := jc.GetJobStatus("job-name", 1*time.Millisecond, 1)
	assert.Error(t, err)
	assert.Equal(t, js, "")
	assert.Contains(t, err.Error(), "Job job-name has not been finished after specified delay")
}

func TestJenkinsClient_GetJobStatus_ShouldFailOnNotbuilt(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "notbuilt",
	}
	httpmock.RegisterResponder("GET", "j-url/job/job-name/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))

	js, err := jc.GetJobStatus("job-name", 1*time.Millisecond, 1)
	assert.Error(t, err)
	assert.Equal(t, js, "")
	assert.Contains(t, err.Error(), "Job job-name has not been finished after specified delay")
}
