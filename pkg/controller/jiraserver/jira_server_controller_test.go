package jiraserver

import (
	"context"
	"os"
	"strings"
	"testing"

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

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

func TestReconcileJiraServer_Reconcile_ShouldPassNotFound(t *testing.T) {
	j := &codebaseApi.JiraServer{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, j)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileJiraServer_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	if !strings.Contains(err.Error(), "no kind is registered for the type v1.JiraServer") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
	assert.False(t, res.Requeue)
}

func TestReconcileJiraServer_Reconcile_ShouldFailInitJiraClientWithSecretNotFound(t *testing.T) {
	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewJira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, j)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "couldn't get secret jira-secret") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraServer_Reconcile_ShouldFailToCreateNewJiraClient(t *testing.T) {
	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewJira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
			ApiUrl:         "htt\\p://example.com",
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
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j, s).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "couldn't create Jira client") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraServer_Reconcile_ShouldPass(t *testing.T) {
	os.Setenv("ASSETS_DIR", "../../../build")
	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewJira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
			ApiUrl:         "j-api",
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
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, j, &edpCompApi.EDPComponent{})
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j, s).Build()

	httpmock.Reset()
	httpmock.Activate()

	ju := jira.User{
		Name: "user",
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/myself",
		httpmock.NewJsonResponderOrPanic(200, &ju))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileJiraServer_Reconcile_ShouldFailToCreateEDPComponent(t *testing.T) {
	os.Setenv("ASSETS_DIR", "../../../build")
	j := &codebaseApi.JiraServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewJira",
			Namespace: "namespace",
		},
		Spec: codebaseApi.JiraServerSpec{
			CredentialName: "jira-secret",
			ApiUrl:         "j-api",
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
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j, s).Build()

	httpmock.Reset()
	httpmock.Activate()

	ju := jira.User{
		Name: "user",
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/myself",
		httpmock.NewJsonResponderOrPanic(200, &ju))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "couldn't create EDP component NewJira") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
