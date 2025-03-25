package platform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetKrciConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name      string
		k8sClient client.Client
		wantErr   require.ErrorAssertionFunc
		want      *KrciConfig
	}{
		{
			name: "should return krci config",
			k8sClient: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      KrciConfigMap,
						Namespace: "default",
					},
					Data: map[string]string{
						"dns_wildcard":                   "test-wildcard",
						"container_registry_type":        "test-type",
						"edp_version":                    "test-version",
						KrciConfigContainerRegistryHost:  "test-registry",
						KrciConfigContainerRegistrySpace: "test-space",
					},
				}).
				Build(),
			wantErr: require.NoError,
			want: &KrciConfig{
				DnsWildcard:                      "test-wildcard",
				ContainerRegistryType:            "test-type",
				EdpVersion:                       "test-version",
				KrciConfigContainerRegistryHost:  "test-registry",
				KrciConfigContainerRegistrySpace: "test-space",
			},
		},
		{
			name: "should return error if config map not found",
			k8sClient: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			wantErr: require.Error,
			want:    nil,
		},
		{
			name: "should return krci config from edp-config for backward compatibility",
			k8sClient: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "edp-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"dns_wildcard":                   "test-wildcard",
						"container_registry_type":        "test-type",
						"edp_version":                    "test-version",
						KrciConfigContainerRegistryHost:  "test-registry",
						KrciConfigContainerRegistrySpace: "test-space",
					},
				}).
				Build(),
			wantErr: require.NoError,
			want: &KrciConfig{
				DnsWildcard:                      "test-wildcard",
				ContainerRegistryType:            "test-type",
				EdpVersion:                       "test-version",
				KrciConfigContainerRegistryHost:  "test-registry",
				KrciConfigContainerRegistrySpace: "test-space",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKrciConfig(context.Background(), tt.k8sClient, "default")
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
