package v2

import (
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// CLIEnabledEnv set to false makes Clone and Push use go-git instead of the git binary.
const CLIEnabledEnv = "GIT_CLI_ENABLED"

// GitProviderFactory creates a Git provider from GitServer and Secret.
// This is a factory pattern to enable dependency injection and mocking in tests.
type GitProviderFactory func(cfg Config) Git

// DefaultGitProviderFactory is the default factory implementation for creating Git providers.
func DefaultGitProviderFactory(gitServer *codebaseApi.GitServer, secret *corev1.Secret) Git {
	return NewGitProviderFactory(NewConfigFromGitServerAndSecret(gitServer, secret))
}

// NewGitProviderFactory returns a CLIProvider, or a GitProvider when CLIEnabled is false.
func NewGitProviderFactory(cfg Config) Git {
	if CLIEnabled() {
		return NewCLIProvider(cfg)
	}

	return NewGitProvider(cfg)
}

// CLIEnabled reports whether Clone and Push run the git binary: unless GIT_CLI_ENABLED parses as false.
func CLIEnabled() bool {
	enabled, err := strconv.ParseBool(os.Getenv(CLIEnabledEnv))

	return err != nil || enabled
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
