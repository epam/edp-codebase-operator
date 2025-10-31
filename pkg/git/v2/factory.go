package v2

import (
	corev1 "k8s.io/api/core/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// GitProviderFactory creates a Git provider from GitServer and Secret.
// This is a factory pattern to enable dependency injection and mocking in tests.
type GitProviderFactory func(cfg Config) Git

// DefaultGitProviderFactory is the default factory implementation for creating GitProvider instances.
func DefaultGitProviderFactory(gitServer *codebaseApi.GitServer, secret *corev1.Secret) Git {
	return NewGitProvider(NewConfigFromGitServerAndSecret(gitServer, secret))
}

func NewGitProviderFactory(cfg Config) Git {
	return NewGitProvider(cfg)
}

func NewConfigFromGitServerAndSecret(gitServer *codebaseApi.GitServer, secret *corev1.Secret) Config {
	return Config{
		SSHKey:      string(secret.Data[util.PrivateSShKeyName]),
		SSHUser:     gitServer.Spec.GitUser,
		SSHPort:     gitServer.Spec.SshPort,
		GitProvider: gitServer.Spec.GitProvider,
		Token:       string(secret.Data[util.GitServerSecretTokenField]),
		Username:    string(secret.Data[util.GitServerSecretUserNameField]),
	}
}
