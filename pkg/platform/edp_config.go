package platform

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const EdpConfigMap = "edp-config"

type EdpConfig struct {
	DnsWildcard string
}

func GetEdpConfig(ctx context.Context, k8sClient client.Client, namespace string) (*EdpConfig, error) {
	edpConfig := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      EdpConfigMap,
	}, edpConfig); err != nil {
		return nil, fmt.Errorf("failed to get edp config: %w", err)
	}

	return mapConfigToEdpConfig(edpConfig), nil
}

func mapConfigToEdpConfig(config *corev1.ConfigMap) *EdpConfig {
	return &EdpConfig{
		DnsWildcard: config.Data["dns_wildcard"],
	}
}
