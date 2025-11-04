package codebase

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// GetRepositoryCredentialsIfExists retrieves the repository credentials from the secret
// referenced in the codebase's CloneRepositoryCredentials field.
func GetRepositoryCredentialsIfExists(
	ctx context.Context,
	codebase *codebaseApi.Codebase,
	k8sClient client.Client,
) (userName, password string, exists bool, err error) {
	secret := &corev1.Secret{}

	err = k8sClient.Get(ctx, types.NamespacedName{
		Namespace: codebase.Namespace,
		Name:      codebase.GetCloneRepositoryCredentialSecret(),
	}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", false, nil
		}

		return "", "", false, fmt.Errorf("failed to get secret with clone repository credentials: %w", err)
	}

	if len(secret.Data["username"]) == 0 {
		return "", "", false, fmt.Errorf("username key is not defined in secret %s", secret.Name)
	}

	if len(secret.Data["password"]) == 0 {
		return "", "", false, fmt.Errorf("password key is not defined in secret %s", secret.Name)
	}

	return string(secret.Data["username"]), string(secret.Data["password"]), true, nil
}
