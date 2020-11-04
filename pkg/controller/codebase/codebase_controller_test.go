package codebase

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository/k8s"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository/sql"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestUpdateFinishStatusUpdatesGitStatus(t *testing.T) {
	cr := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "super-edp",
		},
		Status: v1alpha1.CodebaseStatus{
			Git: "pushed",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &cr)
	objs := []runtime.Object{
		&cr,
	}
	client := fake.NewFakeClient(objs...)

	recoCodebase := ReconcileCodebase{client: client}
	err := recoCodebase.updateFinishStatus(&cr)
	assert.NoError(t, err)

	persCR := v1alpha1.Codebase{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Namespace: "super-edp",
		Name:      "codebase",
	}, &persCR)
	assert.NoError(t, err)

	assert.Equal(t, util.StatusFinished, persCR.Status.Status)
	assert.Equal(t, "pushed", persCR.Status.Git)
}

func TestCreateCodebaseRepoK8S(t *testing.T) {
	cr := v1alpha1.Codebase{}
	reco := ReconcileCodebase{}

	repo := reco.createCodebaseRepo(&cr)

	assert.IsType(t, k8s.CodebaseRepository{}, repo)
}

func TestCreateCodebaseRepoSql(t *testing.T) {
	db, _, _ := sqlmock.New()
	cr := v1alpha1.Codebase{}
	reco := ReconcileCodebase{
		db: db,
	}

	repo := reco.createCodebaseRepo(&cr)

	assert.IsType(t, sql.CodebaseRepository{}, repo)
}

func TestCreateCiChainGitlab(t *testing.T) {
	cr := v1alpha1.Codebase{}
	reco := ReconcileCodebase{}
	clientSet := openshift.ClientSet{}

	hand, err := reco.getCiChain(&cr, &clientSet, nil)

	assert.NoError(t, err)
	assert.NotNil(t, hand)
}

func TestCreateCiChainVCS(t *testing.T) {
	cr := v1alpha1.Codebase{Spec: v1alpha1.CodebaseSpec{
		CiTool: util.GitlabCi,
	}}
	reco := ReconcileCodebase{}
	clientSet := openshift.ClientSet{}

	hand, err := reco.getCiChain(&cr, &clientSet, nil)

	assert.NoError(t, err)
	assert.NotNil(t, hand)
}
