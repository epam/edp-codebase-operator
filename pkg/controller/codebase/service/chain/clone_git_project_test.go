package chain

import (
	"context"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestSetIntermediateSuccessFields(t *testing.T) {
	cr := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codebase",
			Namespace: "super-edp",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &cr)
	objs := []runtime.Object{
		&cr,
	}
	client := fake.NewFakeClient(objs...)
	cs := openshift.ClientSet{
		Client: client,
	}
	handler := CloneGitProject{clientSet: cs}

	err := handler.setIntermediateSuccessFields(&cr, v1alpha1.JenkinsConfiguration)

	assert.NoError(t, err)

	persCR := v1alpha1.Codebase{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Namespace: "super-edp",
		Name:      "codebase",
	}, &persCR)
	assert.NoError(t, err)

	assert.Equal(t, v1alpha1.JenkinsConfiguration, persCR.Status.Action)
	assert.Equal(t, v1alpha1.Success, persCR.Status.Result)
}
