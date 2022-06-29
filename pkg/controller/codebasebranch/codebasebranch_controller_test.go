package codebasebranch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/go-logr/logr"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	edpComponentApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestReconcileCodebaseBranch_Reconcile_ShouldPassNotFoundCR(t *testing.T) {
	c := &codebaseApi.CodebaseBranch{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1.Codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailGetCodebase(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	c := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "Unable to get Codebase ") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailDeleteCodebasebranch(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	c := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	cb := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "Unable to remove codebasebranch NewCodebaseBranch") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileCodebaseBranch_Reconcile_ShouldPassDeleteCodebasebranch(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
		Spec: codebaseApi.CodebaseSpec{
			CiTool: util.GitlabCi,
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldPassWithDeleteJobFailure(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}
	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available: true,
		},
	}

	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-admin-token",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("j-user"),
			"password": []byte("j-token"),
		},
	}
	js := &jenkinsApi.Jenkins{}
	jl := &jenkinsApi.JenkinsList{
		Items: []jenkinsApi.Jenkins{
			{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "jenkins",
					Namespace: "namespace",
					Annotations: map[string]string{
						"edp.epam.com/admin-token": "fake-admin-token",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, cb, jf, js, jl)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, jf, s, js, jl).Build()

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "red",
		LastBuild: gojenkins.JobBuild{
			Number: 10,
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/job/NewCodebase/job/Delete-release-NewCodebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/job/NewCodebase/job/Delete-release-NewCodebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))
	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.namespace:8080/job/NewCodebase/job/Delete-release-NewCodebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.Equal(t, res.RequeueAfter, 10*time.Second)
	cResp := &codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		cResp)
	assert.NoError(t, err)
	assert.Equal(t, cResp.Status.FailureCount, int64(1))
}

func TestReconcileCodebaseBranch_Reconcile_ShouldPassWithCreatingCIS(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			BranchName:   "master",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}
	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available: true,
		},
	}

	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-admin-token",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("j-user"),
			"password": []byte("j-token"),
		},
	}
	js := &jenkinsApi.Jenkins{}
	jl := &jenkinsApi.JenkinsList{
		Items: []jenkinsApi.Jenkins{
			{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "jenkins",
					Namespace: "namespace",
					Annotations: map[string]string{
						"edp.epam.com/admin-token": "fake-admin-token",
					},
				},
			},
		},
	}

	ec := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "docker-registry",
			Namespace: "namespace",
		},
		Spec: edpComponentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}
	cis := &codebaseApi.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, cb, jf, js, jl, ec, cis)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, jf, s, js, jl, ec, cis).Build()

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "blue",
		LastBuild: gojenkins.JobBuild{
			Number: 10,
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/job/NewCodebase/job/Create-release-NewCodebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/job/NewCodebase/job/Create-release-NewCodebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))
	httpmock.RegisterResponder("GET", "http://jenkins.namespace:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.namespace:8080/job/NewCodebase/job/Create-release-NewCodebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	cResp := &codebaseApi.CodebaseImageStream{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCodebase-master",
			Namespace: "namespace",
		},
		cResp)
	assert.NoError(t, err)
	assert.Equal(t, cResp.Spec.ImageName, "stub-url/namespace/NewCodebase")

	gotCodebaseBranch := &codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(), types.NamespacedName{
		Name:      "NewCodebaseBranch",
		Namespace: "namespace",
	}, gotCodebaseBranch)
	if err != nil {
		t.Fatal(err)
	}
	expectedLabels := map[string]string{
		codebasebranch.LabelCodebaseName: "NewCodebase",
	}
	assert.Equal(t, expectedLabels, gotCodebaseBranch.GetLabels())
}

func TestReconcileCodebaseBranch_Reconcile_ShouldRequeueWithCodebaseNotReady(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Status: "done",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, cb, jf)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, jf).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.Equal(t, res.RequeueAfter, 5*time.Second)

	gotCodebaseBranch := &codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(), types.NamespacedName{
		Name:      "NewCodebaseBranch",
		Namespace: "namespace",
	}, gotCodebaseBranch)
	if err != nil {
		t.Fatal(err)
	}
	expectedLabels := map[string]string{
		codebasebranch.LabelCodebaseName: "NewCodebase",
	}
	assert.Equal(t, expectedLabels, gotCodebaseBranch.GetLabels())
}

func TestReconcileCodebaseBranch_Reconcile_ShouldInitBuildForEDPVersioning(t *testing.T) {
	t.Parallel()

	namespace := "test-namespace"
	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: util.VersioningTypeEDP,
			},
		},
	}
	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: namespace,
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, cb, jf)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, jf).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebaseBranch",
			Namespace: namespace,
		},
	}

	r := ReconcileCodebaseBranch{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, res.Requeue)

	gotCodebaseBranch := codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.Background(), types.NamespacedName{
		Name:      "NewCodebaseBranch",
		Namespace: namespace,
	}, &gotCodebaseBranch)

	require.NoError(t, err)

	expectedBuildNumber := "0"

	assert.NotNil(t, gotCodebaseBranch.Status.Build)
	assert.Equal(t, &expectedBuildNumber, gotCodebaseBranch.Status.Build)
}
