package codebase

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	chainMocks "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

type ControllerTestSuite struct {
	suite.Suite
	scheme *runtime.Scheme
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

func (s *ControllerTestSuite) SetupTest() {
	err := os.Setenv("WORKING_DIR", "/tmp/1")
	require.NoError(s.T(), err)

	s.scheme = runtime.NewScheme()
	assert.NoError(s.T(), codebaseApi.AddToScheme(s.scheme))
	assert.NoError(s.T(), jenkinsApi.AddToScheme(s.scheme))
	s.scheme.AddKnownTypes(coreV1.SchemeGroupVersion, &coreV1.ConfigMap{}, &coreV1.Secret{})
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldPassNotFound() {
	c := &codebaseApi.Codebase{}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
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

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered for the type v1.Codebase")
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldFailDeleteCodebase() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, &codebaseApi.Codebase{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.Error(t, err)
	assert.False(t, res.Requeue)
	assert.Contains(t, err.Error(), "an error has occurred while trying to delete codebase")
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldPassOnInvalidCodebase() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldFailOnCreateStrategy() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Create,
			Lang:     "go",
		},
	}

	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 10*time.Second)

	cResp := &codebaseApi.Codebase{}
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
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Create,
			Lang:     "java",
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("templates_pushed"),
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
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
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase-codebase",
			Namespace: "namespace",
		},
	}
	secret := &coreV1.Secret{}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c, cm, gs, jf, secret).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, time.Duration(0))
}

func (s *ControllerTestSuite) TestReconcileCodebase_Reconcile_ShouldDeleteCodebase() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
	}
	cbl := &codebaseApi.CodebaseBranchList{}
	jfl := &jenkinsApi.JenkinsFolderList{}

	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c, cbl, jfl).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
	}

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	t := s.T()
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategy() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: util.ImportStrategy,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: scheme,
	}
	ch, err := r.getStrategyChain(c)

	t := s.T()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategyGitLab() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: util.ImportStrategy,
			CiTool:   util.GitlabCi,
		},
	}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}
	ch, err := r.getStrategyChain(c)

	t := s.T()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassCloneStrategyTekton() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Clone,
			CiTool:   util.Tekton,
		},
	}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}
	ch, err := r.getStrategyChain(c)

	t := s.T()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func (s *ControllerTestSuite) TestReconcileCodebase_getStrategyChain_ShouldPassImportStrategyTekton() {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCodebase",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: codebaseApi.Import,
			CiTool:   util.Tekton,
		},
	}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(c).Build()
	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
	}
	ch, err := r.getStrategyChain(c)

	t := s.T()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func (s *ControllerTestSuite) TestPostpone() {
	ctx := context.Background()
	c := codebaseApi.Codebase{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Codebase",
			APIVersion: "v2.edp.epam.com/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:            "NewCodebase",
			Namespace:       "namespace",
			ResourceVersion: "1",
		},
		Spec: codebaseApi.CodebaseSpec{
			Strategy: util.ImportStrategy,
			CiTool:   util.GitlabCi,
			Type:     "application",
			Lang:     "java",
		},
	}
	fakeCl := fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(&c).Build()

	handlerMock := chainMocks.CodebaseHandler{}

	cloneCb := c.DeepCopy()
	cloneCb.ResourceVersion = "3"
	cloneCb.Labels = map[string]string{"app.edp.epam.com/codebaseType": "application"}
	cloneCb.Finalizers = []string{"codebase.operator.finalizer.name", "foregroundDeletion"}
	handlerMock.On("ServeRequest", ctx, cloneCb).Return(chain.PostponeError{Timeout: time.Second})

	r := ReconcileCodebase{
		client: fakeCl,
		log:    logr.Discard(),
		scheme: s.scheme,
		chainGetter: func(cr *codebaseApi.Codebase) (handler.CodebaseHandler, error) {
			return &handlerMock, nil
		},
	}

	res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, time.Second)

	handlerMock.AssertExpectations(s.T())
}
