package codebase

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileCodebase_SetupWithManager(t *testing.T) {
	os.Setenv("DB_ENABLED", "false")

	r := NewReconcileCodebase(nil, nil, logr.DiscardLogger{})
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

func TestReconcileCodebase_Reconcile_ShouldPassNotFound(t *testing.T) {
	c := &v1alpha1.Codebase{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebase_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
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

func TestReconcileCodebase_Reconcile_ShouldFailDeleteCodebase(t *testing.T) {
	os.Setenv("WORKING_DIR", "/tmp/1")
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	if !strings.Contains(err.Error(), "an error has occurred while trying to delete codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcileCodebase_Reconcile_ShouldPassOnInvalidCodebase(t *testing.T) {
	os.Setenv("WORKING_DIR", "/tmp/1")
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebase_Reconcile_ShouldFailOnCreateStrategy(t *testing.T) {
	os.Setenv("WORKING_DIR", "/tmp/1")
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Strategy: v1alpha1.Create,
			Lang:     "go",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 10*time.Second)

	cResp := &v1alpha1.Codebase{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		cResp)
	assert.NoError(t, err)
	assert.Equal(t, cResp.Status.FailureCount, int64(1))
}

func TestReconcileCodebase_Reconcile_ShouldPassOnJavaCreateStrategy(t *testing.T) {
	os.Setenv("WORKING_DIR", "/tmp/1")
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Strategy: v1alpha1.Create,
			Lang:     "java",
		},
		Status: v1alpha1.CodebaseStatus{
			Git: *util.GetStringP("templates_pushed"),
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edp-config",
			Namespace: "namespace",
		},
		Data: map[string]string{
			"vcs_integration_enabled":  "false",
			"perf_integration_enabled": "false",
			"dns_wildcard":             "dns",
			"edp_name":                 "edp-name",
			"edp_version":              "2.2.2",
			"vcs_group_name_url":       "https://gitlab.example.com/backup",
			"vcs_ssh_port":             "22",
			"vcs_tool_name":            "gitlab",
		},
	}
	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GitServerSpec{
			SshPort: 22,
		},
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
	}
	s := &coreV1.Secret{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs, jf)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cm, gs, jf, s).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, time.Duration(0))
}

func TestReconcileCodebase_Reconcile_ShouldDeleteCodebase(t *testing.T) {
	os.Setenv("WORKING_DIR", "/tmp/1")
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
	}
	cbl := &v1alpha1.CodebaseBranchList{}
	jfl := &jenkinsv1alpha1.JenkinsFolderList{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cbl, jfl)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cbl, jfl).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategy(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Strategy: util.ImportStrategy,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}
	ch, err := r.getStrategyChain(c)
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategyGitLab(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Strategy: util.ImportStrategy,
			CiTool:   util.GitlabCi,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}
	ch, err := r.getStrategyChain(c)
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestReconcileCodebase_getStrategyChain_ShouldPassWothDb(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	c := &v1alpha1.Codebase{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
		db:     db,
	}
	repo := r.createCodebaseRepo(c)
	assert.NotNil(t, repo)
}
