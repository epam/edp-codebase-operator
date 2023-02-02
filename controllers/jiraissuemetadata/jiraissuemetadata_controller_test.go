package jiraissuemetadata

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/go-logr/logr"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestReconcileJiraIssueMetadata_Reconcile_ShouldPassNotFound(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.JiraIssueMetadata") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailToSetOwnerReference(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.Codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailJiraISDisabled(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &codebaseApi.Codebase{
		TypeMeta: metaV1.TypeMeta{
			Kind: "Codebase",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "codebase codebase has disabled jira integration. skip JiraIssueMetadata JIM reconcilation") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailGetJira(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &codebaseApi.Codebase{
		TypeMeta: metaV1.TypeMeta{
			Kind: "Codebase",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c, &codebaseApi.JiraServer{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "jiraservers.v2.edp.epam.com \"jira\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.False(t, res.Requeue)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldPassJiraFoundButUnavailable(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Status: codebaseApi.JiraServerStatus{
			Available: false,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c, j)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 500*time.Millisecond)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailInitJiraClient(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-sercret",
		},
		Status: codebaseApi.JiraServerStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c, j, &coreV1.Secret{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)

	if !strings.Contains(err.Error(), "failed to get secret jira-sercret") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailToCreateChain(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
		},
		Status: codebaseApi.JiraServerStatus{
			Available: true,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j, s).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)

	if !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldFailServeRequest(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
			Payload:      "{}",
			Tickets:      []string{"T1", "T2"},
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
		},
		Status: codebaseApi.JiraServerStatus{
			Available: true,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ist, c, j, s).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 500*time.Millisecond)
}

func TestReconcileJiraIssueMetadata_Reconcile_ShouldPass(t *testing.T) {
	ist := &codebaseApi.JiraIssueMetadata{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "JIM",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraIssueMetadataSpec{
			CodebaseName: "codebase",
			Payload:      `{"issuesLinks": [{"ticket":"fake-issueId", "title":"fake-title", "url":"fake-url"}]}`,
			Tickets:      []string{"T1"},
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "codebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			JiraServer: util.GetStringP("jira"),
		},
	}

	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
		},
		Status: codebaseApi.JiraServerStatus{
			Available: true,
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jira-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, ist, c, j)
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

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "JIM",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraIssueMetadata{
		client: fakeCl,
		scheme: scheme,
		log:    logr.Discard(),
	}

	rec, err := r.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	assert.False(t, rec.Requeue)

	duration, err := time.ParseDuration(defaultReconcilePeriod + "m")
	if err != nil {
		t.Fatal("failed to parse time")
	}

	assert.Equal(t, rec.RequeueAfter, duration)
}
