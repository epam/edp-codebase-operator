package cdstagedeploy

import (
	"context"
	"testing"
	"time"

	v1alpha1Stage "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileCDStageDeploy_Reconcile_ShouldPass(t *testing.T) {
	j := &jenkinsApi.Jenkins{}
	jl := &jenkinsApi.JenkinsList{
		Items: []jenkinsApi.Jenkins{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jenkins",
					Namespace: "namespace",
				},
			},
		},
	}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{}
	s := &v1alpha1Stage.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	cdsd := &v1alpha1.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: jenkinsApi.Tag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(metav1.SchemeGroupVersion, jl, j)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, cdsd)
	scheme.AddKnownTypes(v1alpha1Stage.SchemeGroupVersion, s)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, s, jcdsd, j, jl).Build()

	//request
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

func TestReconcileCDStageDeploy_Reconcile_ShouldFailtoGetCDStageDeploy(t *testing.T) {

	scheme := runtime.NewScheme()

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	//request
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
	assert.Contains(t, err.Error(), "no kind is registered for the type v1alpha1.CDStageDeploy in scheme")
}

func TestReconcileCDStageDeploy_Reconcile_ShouldPassWithNoFoundCDStageDeploy(t *testing.T) {
	cdsd := &v1alpha1.CDStageDeploy{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd).Build()

	//request
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
	cdsd := &v1alpha1.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd).Build()

	//request
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
	assert.Contains(t, err.Error(), "no kind is registered for the type v1alpha1.Stage in scheme")
}

func TestReconcileCDStageDeploy_Reconcile_ShouldFailServeRequest(t *testing.T) {
	cdsd := &v1alpha1.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: jenkinsApi.Tag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	s := &v1alpha1Stage.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1Stage.SchemeGroupVersion, s)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, s).Build()

	//request
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
	cdsd := &v1alpha1.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: v1alpha1.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: jenkinsApi.Tag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	s := &v1alpha1Stage.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1Stage.SchemeGroupVersion, s)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, cdsd)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, jcdsd, s).Build()

	//request
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

	cdsdResp := &v1alpha1.CDStageDeploy{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		cdsdResp)
	assert.NoError(t, err)
	assert.Equal(t, cdsdResp.Status.FailureCount, int64(1))
}
