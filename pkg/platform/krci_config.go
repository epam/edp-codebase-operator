package platform

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const KrciConfigMap = "krci-config"

type KrciConfig struct {
	DnsWildcard string
}

func GetKrciConfig(ctx context.Context, k8sClient client.Client, namespace string) (*KrciConfig, error) {
	KrciConfig := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      KrciConfigMap,
	}, KrciConfig); err != nil {
		return nil, fmt.Errorf("failed to get krci config: %w", err)
	}

	return mapConfigToKrciConfig(KrciConfig), nil
}

func mapConfigToKrciConfig(config *corev1.ConfigMap) *KrciConfig {
	return &KrciConfig{
		DnsWildcard: config.Data["dns_wildcard"],
	}
}
