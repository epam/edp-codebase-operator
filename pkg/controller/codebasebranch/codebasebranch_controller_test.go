package codebasebranch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	edpV1alpha1 "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestReconcileCodebaseBranch_Reconcile_ShouldPassNotFoundCR(t *testing.T) {
	c := &v1alpha1.CodebaseBranch{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
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
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.Codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileCodebaseBranch_Reconcile_ShouldFailGetCodebase(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	if err != nil {
		t.Fatal(err)
	}
	c := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
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
	c := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	cb := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cb)
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
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
		Spec: v1alpha1.CodebaseSpec{
			CiTool: util.GitlabCi,
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cb)
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
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
	}
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}
	jf := &jenkinsv1alpha1.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
			Available: true,
		},
	}

	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-admin-token",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("j-user"),
			"password": []byte("j-token"),
		},
	}
	js := &jenkinsv1alpha1.Jenkins{}
	jl := &jenkinsv1alpha1.JenkinsList{
		Items: []jenkinsv1alpha1.Jenkins{
			{
				ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cb, jf, js, jl)
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
	cResp := &v1alpha1.CodebaseBranch{}
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
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
			BranchName:   "master",
		},
	}
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}
	jf := &jenkinsv1alpha1.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
			Available: true,
		},
	}

	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-admin-token",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("j-user"),
			"password": []byte("j-token"),
		},
	}
	js := &jenkinsv1alpha1.Jenkins{}
	jl := &jenkinsv1alpha1.JenkinsList{
		Items: []jenkinsv1alpha1.Jenkins{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jenkins",
					Namespace: "namespace",
					Annotations: map[string]string{
						"edp.epam.com/admin-token": "fake-admin-token",
					},
				},
			},
		},
	}

	ec := &edpV1alpha1.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docker-registry",
			Namespace: "namespace",
		},
		Spec: edpV1alpha1.EDPComponentSpec{
			Url: "stub-url",
		},
	}
	cis := &v1alpha1.CodebaseImageStream{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cb, jf, js, jl, ec, cis)
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
	cResp := &v1alpha1.CodebaseImageStream{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCodebase-master",
			Namespace: "namespace",
		},
		cResp)
	assert.NoError(t, err)
	assert.Equal(t, cResp.Spec.ImageName, "stub-url/namespace/NewCodebase")

	gotCodebaseBranch := &v1alpha1.CodebaseBranch{}
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
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebaseBranch",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "NewCodebase",
		},
		Status: v1alpha1.CodebaseBranchStatus{
			Status: "done",
		},
	}
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cb, jf)
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

	gotCodebaseBranch := &v1alpha1.CodebaseBranch{}
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
