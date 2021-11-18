package jiraissuemetadata

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-logr/logr"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileJiraIssueMetadata_SetupWithManager(t *testing.T) {
	r := NewReconcileJiraIssueMetadata(nil, nil, logr.DiscardLogger{})
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{MetricsBindAddress: "0"})
	if err != nil {
		t.Fatal(err)
	}

	err = r.SetupWithManager(mgr)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "no kind is registered for the type") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldPassNotFound(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.JiraIssueMetadata") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailToSetOwnerReference(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1alpha1.Codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailJiraISDisabled(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &v1alpha1.Codebase{
		TypeMeta: metav1.TypeMeta{
			Kind: "Codebase",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "codebase codebase has disabled jira integration. skip JiraIssueMetadata JIM reconcilation") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailGetJira(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &v1alpha1.Codebase{
		TypeMeta: metav1.TypeMeta{
			Kind: "Codebase",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c, &v1alpha1.JiraServer{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "jiraservers.v2.edp.epam.com \"jira\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldPassJiraFoundButUnavailable(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &v1alpha1.JiraServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Status: v1alpha1.JiraServerStatus{
			Available: false,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c, j)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 500*time.Millisecond)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailInitJiraClient(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &v1alpha1.JiraServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraServerSpec{
			CredentialName: "jira-sercret",
		},
		Status: v1alpha1.JiraServerStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c, j, &coreV1.Secret{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "couldn't get secret jira-sercret") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailToCreateChain(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &v1alpha1.JiraServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraServerSpec{
			CredentialName: "jira-secret",
		},
		Status: v1alpha1.JiraServerStatus{
			Available: true,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j, s).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailServeRequest(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
			Payload:      "{}",
			Tickets:      []string{"T1", "T2"},
		},
	}

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &v1alpha1.JiraServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraServerSpec{
			CredentialName: "jira-secret",
		},
		Status: v1alpha1.JiraServerStatus{
			Available: true,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j, s).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 500*time.Millisecond)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldPass(t *testing.T) {
	ist := &v1alpha1.JiraIssueMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
			Payload:      `{"issuesLinks": [{"ticket":"fake-issueId", "title":"fake-title", "url":"fake-url"}]}`,
			Tickets:      []string{"T1"},
		},
	}

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &v1alpha1.JiraServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: v1alpha1.JiraServerSpec{
			CredentialName: "jira-secret",
		},
		Status: v1alpha1.JiraServerStatus{
			Available: true,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, ist, c, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j, s).Build()

	httpmock.Reset()
	httpmock.Activate()
	ji := jira.Issue{
		Fields: &jira.IssueFields{
			Project: jira.Project{
				ID: "1",
			},
			Type: jira.IssueType{
				Name: "Bug",
			},
		},
	}

	issueMetadata := &jira.CreateMetaInfo{
		Projects: []*jira.MetaProject{
			{
				Id: "1",
				IssueTypes: []*jira.MetaIssueType{
					{
						Name: "Bug",
					},
				},
			},
		},
	}

	jm := jira.CreateMetaInfo{
		Expand:   "expand",
		Projects: issueMetadata.Projects,
	}

	httpmock.RegisterResponder("GET", "/rest/api/2/issue/T1",
		httpmock.NewJsonResponderOrPanic(200, &ji))

	httpmock.RegisterResponder("GET", "/rest/api/2/issue/createmeta?expand=projects.issuetypes.fields",
		httpmock.NewJsonResponderOrPanic(200, &jm))

	httpmock.RegisterResponder("PUT", "/rest/api/2/issue/T1",
		httpmock.NewJsonResponderOrPanic(200, &ji))

	jrl := jira.RemoteLink{}

	httpmock.RegisterResponder("POST", "/rest/api/2/issue/fake-issueId/remotelink",
		httpmock.NewJsonResponderOrPanic(200, &jrl))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.DiscardLogger{},
	}

	rec, err := r.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	assert.False(t, rec.Requeue)
	duration, err := time.ParseDuration(defaultReconcilePeriod + "m")
	if err != nil {
		t.Fatal("Unable to parse time")
	}
	assert.Equal(t, rec.RequeueAfter, duration*time.Minute)
}
