package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func TestCodebaseBranchService_TriggerReleaseJob(t *testing.T) {
	cb := codebaseApi.CodebaseBranch{
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "codebase",
			ReleaseJobParams: map[string]string{
				"codebaseName": "RELEASE_NAME",
				"fromCommit":   "COMMIT_ID",
				"gitServer":    "GIT_SERVER",
			},
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Status: model.StatusInit,
		},
	}
	c := codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
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

	jrsp := gojenkins.JobResponse{
		Color: jenkinsJobSuccessStatus,
	}

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

func TestCodebaseBranchService_TriggerReleaseJobFailedJob(t *testing.T) {
	codebaseBranch := codebaseApi.CodebaseBranch{
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "codebase",
			ReleaseJobParams: map[string]string{
				"codebaseName": "RELEASE_NAME",
				"fromCommit":   "COMMIT_ID",
				"gitServer":    "GIT_SERVER",
			},
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Status: model.StatusInit,
		},
	}
	codebase := codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "codebase",
		},
	}

	secret := coreV1.Secret{}
	jenkins := jenkinsApi.Jenkins{}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &codebaseBranch, &jenkins, &jenkinsApi.JenkinsList{}, &codebase)
	cl := fake.NewClientBuilder().WithRuntimeObjects(&codebaseBranch, &jenkins, &secret, &codebase).Build()
	svc := CodebaseBranchServiceProvider{
		Client: cl,
	}

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jobResponse := gojenkins.JobResponse{
		Color: "red",
	}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jobResponse))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Create-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	err := svc.TriggerReleaseJob(&codebaseBranch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create release, job status: red")
}

func TestCodebaseBranchService_TriggerDeletionJob(t *testing.T) {
	cb := codebaseApi.CodebaseBranch{
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "codebase",
		},
	}

	c := codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   jenkinsJobSuccessStatus,
		LastBuild: gojenkins.JobBuild{
			Number: 10,
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	if err := svc.TriggerDeletionJob(&cb); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestCodebaseBranchService_TriggerDeletionJobFailed(t *testing.T) {
	cb := codebaseApi.CodebaseBranch{
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "codebase",
		},
	}

	c := codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
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

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "red",
		LastBuild: gojenkins.JobBuild{
			Number: 10,
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	err := svc.TriggerDeletionJob(&cb)
	assert.Error(t, err)

	if errors.Cause(err) != JobFailedError(err.Error()) {
		t.Fatal("wrong error returned")
	}
}

func TestCodebaseBranchServiceProvider_AppendVersionToTheHistorySlice(t *testing.T) {
	version := "0-0-1-SNAPSHOT"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			Version: &version,
		},
		Status: codebaseApi.CodebaseBranchStatus{
			VersionHistory: []string{"0-0-0-SNAPSHOT"},
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.AppendVersionToTheHistorySlice(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Equal(t, len(cbResp.Status.VersionHistory), 2)
	assert.Equal(t, cbResp.Status.VersionHistory[1], version)
}

func TestCodebaseBranchServiceProvider_ResetBranchBuildCounter(t *testing.T) {
	b := "100"
	zb := "0"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Build: &b,
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.ResetBranchBuildCounter(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Build, &zb)
}

func TestCodebaseBranchServiceProvider_ResetBranchSuccessBuildCounter(t *testing.T) {
	b := "100"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			LastSuccessfulBuild: &b,
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.ResetBranchSuccessBuildCounter(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Nil(t, cbResp.Status.LastSuccessfulBuild)
}
