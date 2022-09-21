package cdstagedeploy

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func TestReconcileCDStageDeploy_Reconcile_ShouldPass(t *testing.T) {
	j := &jenkinsApi.Jenkins{}
	jl := &jenkinsApi.JenkinsList{
		Items: []jenkinsApi.Jenkins{
			{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "jenkins",
					Namespace: "namespace",
				},
			},
		},
	}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: jenkinsApi.Tag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(metaV1.SchemeGroupVersion, jl, j)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, cdsd)
	scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, s)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, s, jcdsd, j, jl).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	r := ReconcileCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCDStageDeploy_Reconcile_ShouldFailToGetCDStageDeploy(t *testing.T) {
	scheme := runtime.NewScheme()

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	r := ReconcileCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	assert.Contains(t, err.Error(), "no kind is registered for the type v1.CDStageDeploy in scheme")
}

func TestReconcileCDStageDeploy_Reconcile_ShouldPassWithNoFoundCDStageDeploy(t *testing.T) {
	cdsd := &codebaseApi.CDStageDeploy{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	r := ReconcileCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue)
}

func TestReconcileCDStageDeploy_Reconcile_ShouldFailSetOwnerRef(t *testing.T) {
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
			DeletionTimestamp: &metaV1.Time{
				Time: metaV1.Now().Time,
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	r := ReconcileCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	assert.Contains(t, err.Error(), "no kind is registered for the type v1.Stage in scheme")
}

func TestReconcileCDStageDeploy_Reconcile_ShouldFailServeRequest(t *testing.T) {
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: jenkinsApi.Tag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, s).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	r := ReconcileCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.Error(t, err)
	assert.False(t, res.Requeue)
	assert.Contains(t, err.Error(), "couldn't get NewCDStageDeploy cd stage jenkins deployment")
}

func TestReconcileCDStageDeploy_Reconcile_ShouldFailServeRequestWithExistingCR(t *testing.T) {
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: jenkinsApi.Tag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, cdsd)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, jcdsd, s).Build()

	// request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	r := ReconcileCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
		scheme: scheme,
	}

	res, err := r.Reconcile(context.TODO(), req)

	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, 10*time.Second)

	cdsdResp := &codebaseApi.CDStageDeploy{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		cdsdResp)
	assert.NoError(t, err)
	assert.Equal(t, cdsdResp.Status.FailureCount, int64(1))
}
