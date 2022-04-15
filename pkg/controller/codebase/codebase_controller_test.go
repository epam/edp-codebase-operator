package codebase

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
)

type ControllerTestSuite struct {
	suite.Suite
	scheme *runtime.Scheme
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

func (s *ControllerTestSuite) SetupTest() {
	os.Setenv("WORKING_DIR", "/tmp/1")
	s.scheme = runtime.NewScheme()
	assert.NoError(s.T(), v1alpha1.AddToScheme(s.scheme))
	assert.NoError(s.T(), jenkinsv1alpha1.AddToScheme(s.scheme))
	s.scheme.AddKnownTypes(coreV1.SchemeGroupVersion, &coreV1.ConfigMap{}, &coreV1.Secret{})
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldPassNotFound() {
	c := &v1alpha1.Codebase{}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()

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
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldFailNotFound() {
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

	t := s.T()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered for the type v1alpha1.Codebase")
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldFailDeleteCodebase() {
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Codebase{})
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

	t := s.T()
	assert.Error(t, err)
	assert.False(t, res.Requeue)
	assert.Contains(t, err.Error(), "an error has occurred while trying to delete codebase")
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldPassOnInvalidCodebase() {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()

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
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldFailOnCreateStrategy() {
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

	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()

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
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
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

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldPassOnJavaCreateStrategy() {
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
	secret := &coreV1.Secret{}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c, cm, gs, jf, secret).Build()

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
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, time.Duration(0))
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldDeleteCodebase() {
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

	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c, cbl, jfl).Build()

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
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategy() {
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

	t := s.T()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategyGitLab() {
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
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: s.scheme,
	}
	ch, err := r.getStrategyChain(c)

	t := s.T()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassWothDb() {
	db, _, _ := sqlmock.New()
	defer db.Close()

	c := &v1alpha1.Codebase{}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: s.scheme,
		db:     db,
	}
	repo := r.createCodebaseRepo(c)

	t := s.T()
	assert.NotNil(t, repo)
}

func (s *ControllerTestSuite) TestPostpone() {
	c := v1alpha1.Codebase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Codebase",
			APIVersion: "v2.edp.epam.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "NewCodebase",
			Namespace:       "namespace",
			ResourceVersion: "1",
		},
		Spec: v1alpha1.CodebaseSpec{
			Strategy: util.ImportStrategy,
			CiTool:   util.GitlabCi,
			Lang:     "java",
		},
	}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(&c).Build()

	handlerMock := handler.Mock{}

	cloneCb := c.DeepCopy()
	cloneCb.ResourceVersion = "2"
	cloneCb.Finalizers = []string{"codebase.operator.finalizer.name", "foregroundDeletion"}
	handlerMock.On("ServeRequest", cloneCb).Return(chain.PostponeError{Timeout: time.Second})
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: s.scheme,
		chainGetter: func(cr *v1alpha1.Codebase) (handler.CodebaseHandler, error) {
			return &handlerMock, nil
		},
	}

	res, err := r.Reconcile(context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, time.Second)

	handlerMock.AssertExpectations(s.T())
}
