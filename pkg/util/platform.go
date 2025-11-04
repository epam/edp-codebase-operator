package util

import (
	"context"
	"fmt"
	"os"
	"strconv"

	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
)

const (
	watchNamespaceEnvVar = "WATCH_NAMESPACE"
	debugModeEnvVar      = "DEBUG_MODE"
)

func GetGitServer(c client.Client, name, namespace string) (*model.GitServer, error) {
	gitReq, err := getGitServerCR(c, name, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get %v Git Server CR: %w", name, err)
	}

	gs := model.ConvertToGitServer(gitReq)

	return gs, nil
}

func getGitServerCR(c client.Client, name, namespace string) (*codebaseApi.GitServer, error) {
	log.Info("Start fetching GitServer resource from k8s", "name", name, "namespace", namespace)

	instance := &codebaseApi.GitServer{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to find GitServer %v in k8s: %w", name, err)
		}

		return nil, fmt.Errorf("failed to get GitServer %v: %w", name, err)
	}

	log.Info("Git Server instance has been received", "name", name)

	return instance, nil
}

func GetSecret(c client.Client, secretName, namespace string) (*coreV1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", secretName, "namespace", namespace)

	secret := &coreV1.Secret{}

	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %v: %w", secretName, err)
	}

	log.Info("Secret has been fetched", "secret name", secretName, "namespace", namespace)

	return secret, nil
}

// GetWatchNamespace returns the namespace the operator should be watching for changes.
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}

	return ns, nil
}

// GetDebugMode returns the debug mode value.
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, fmt.Errorf("failed to parse env value as boolean: %w", err)
	}

	return b, nil
}
