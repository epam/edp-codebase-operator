package service

import (
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bndr/gojenkins"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/jarcoal/httpmock"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCodebaseBranchService_TriggerReleaseJob(t *testing.T) {
	cb := v1alpha1.CodebaseBranch{
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "codebase",
			ReleaseJobParams: map[string]string{
				"codebaseName": "RELEASE_NAME",
				"fromCommit":   "COMMIT_ID",
				"gitServer":    "GIT_SERVER",
			},
		},
		Status: v1alpha1.CodebaseBranchStatus{
			Status: model.StatusInit,
		},
	}
	c := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name: "codebase",
		},
	}
	secret := coreV1.Secret{}
	js := jenkinsApi.Jenkins{}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &cb, &js, &jenkinsApi.JenkinsList{}, &c)
	cl := fake.NewClientBuilder().WithRuntimeObjects(&cb, &js, &secret, &c).Build()
	svc := CodebaseBranchServiceProvider{
		Client: cl,
	}

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jrsp := gojenkins.JobResponse{}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Create-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	if err := svc.TriggerReleaseJob(&cb); err != nil {
		t.Fatalf("%+v", err)
	}
}
