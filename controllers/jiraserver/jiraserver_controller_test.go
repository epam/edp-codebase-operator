package jiraserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/andygrunwald/go-jira"
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

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestReconcileJiraServer_Reconcile_ShouldPassNotFound(t *testing.T) {
	j := &codebaseApi.JiraServer{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, j)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)
}

func TestReconcileJiraServer_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.JiraServer") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Equal(t, time.Duration(0), res.RequeueAfter)
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
	scheme.AddKnownTypes(codebaseApi.GroupVersion, j)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)

	if !strings.Contains(err.Error(), "failed to get secret jira-secret") {
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
	scheme.AddKnownTypes(codebaseApi.GroupVersion, j)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(j, s).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)

	if !strings.Contains(err.Error(), "failed to create Jira client") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileJiraServer_Reconcile_ShouldPass(t *testing.T) {
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
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, coreV1.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(j, s).Build()

	httpmock.Reset()
	httpmock.Activate()

	ju := jira.User{
		Name: "user",
	}

	httpmock.RegisterResponder("GET", "/j-api/rest/api/2/myself",
		httpmock.NewJsonResponderOrPanic(200, &ju))

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewJira",
			Namespace: "namespace",
		},
	}

	r := ReconcileJiraServer{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	require.NoError(t, err)
	require.Equal(t, time.Duration(0), res.RequeueAfter)

	jiraServer := &codebaseApi.JiraServer{}
	err = fakeCl.Get(context.Background(), types.NamespacedName{
		Name:      "NewJira",
		Namespace: "namespace",
	}, jiraServer)

	require.NoError(t, err)
	require.Equal(t, statusFinished, jiraServer.Status.Status)
}
