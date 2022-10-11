package helper

import (
	"context"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EDPConfigCM = "edp-config"
	EDPNameKey  = "edp_name"
)

func GetEDPName(ctx context.Context, k8sClient client.Client, namespace string) (*string, error) {
	cm := &coreV1.ConfigMap{}

	err := k8sClient.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      EDPConfigCM,
	}, cm)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 'ConfigMap' resource %q: %w", EDPConfigCM, err)
	}

	r := cm.Data[EDPNameKey]
	if r == "" {
		return nil, fmt.Errorf("there is not key %v in cm %v", EDPNameKey, EDPConfigCM)
	}

	return &r, nil
}
