package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetEDPName_ShouldPass(t *testing.T) {
	ctx := context.Background()
	edpn := "edpName"
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      EDPConfigCM,
			Namespace: "fake-namespace",
		},
		Data: map[string]string{
			EDPNameKey: edpn,
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	n, err := GetEDPName(ctx, fakeCl, "fake-namespace")
	assert.NoError(t, err)
	assert.Equal(t, n, &edpn)
}

func TestGetEDPName_ShouldFailWhenEDPNameNotDefined(t *testing.T) {
	ctx := context.Background()
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      EDPConfigCM,
			Namespace: "fake-namespace",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	n, err := GetEDPName(ctx, fakeCl, "fake-namespace")
	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "there is not key edp_name in cm edp-config")
}

func TestGetEDPName_ShouldFailWhenNotFound(t *testing.T) {
	ctx := context.Background()
	cm := &coreV1.ConfigMap{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	n, err := GetEDPName(ctx, fakeCl, "fake-namespace")
	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "configmaps \"edp-config\" not found")
}
