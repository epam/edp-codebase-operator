package helper

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EDPConfigCM = "edp-config"
	EDPNameKey  = "edp_name"
)

func GetEDPName(client client.Client, namespace string) (*string, error) {
	cm := &v1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      EDPConfigCM,
	}, cm)
	if err != nil {
		return nil, err
	}
	r := cm.Data[EDPNameKey]
	if len(r) == 0 {
		return nil, fmt.Errorf("there is not key %v in cm %v", EDPNameKey, EDPConfigCM)
	}
	return &r, nil
}
