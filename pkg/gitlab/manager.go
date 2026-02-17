package gitlab_ci

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	// GitLabCIDefaultTemplate is the name of the default GitLab CI ConfigMap template.
	GitLabCIDefaultTemplate = "gitlab-ci-default"
	// GitLabCIFileName is the name of the GitLab CI configuration file.
	GitLabCIFileName = ".gitlab-ci.yml"
)

// Manager handles GitLab CI configuration injection.
type Manager interface {
	InjectGitLabCIConfig(ctx context.Context, codebase *codebaseApi.Codebase, workDir string) error
}

type manager struct {
	client client.Client
}

// NewManager creates a new GitLab CI manager.
func NewManager(k8sClient client.Client) Manager {
	return &manager{client: k8sClient}
}

// InjectGitLabCIConfig creates a .gitlab-ci.yml file from ConfigMap template if it doesn't exist.
func (m *manager) InjectGitLabCIConfig(ctx context.Context, codebase *codebaseApi.Codebase, workDir string) error {
	gitlabCIPath := filepath.Join(workDir, GitLabCIFileName)

	// Skip if file already exists
	if _, err := os.Stat(gitlabCIPath); err == nil {
		return nil
	}

	// Get template using hierarchy fallback
	template, err := m.getGitLabCITemplate(ctx, codebase)
	if err != nil {
		return fmt.Errorf("failed to get GitLab CI template: %w", err)
	}

	// Variable substitution — only CodebaseName is replaced; other placeholders stay literal.
	content := strings.ReplaceAll(template, "{{.CodebaseName}}", codebase.Name)

	// Write file
	if err := os.WriteFile(gitlabCIPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", GitLabCIFileName, err)
	}

	return nil
}

// getGitLabCITemplate retrieves the GitLab CI template ConfigMap.
// If the Codebase has the GitLabCITemplateAnnotation, that ConfigMap is used (hard error if missing).
// Otherwise falls back to the "gitlab-ci-default" ConfigMap.
func (m *manager) getGitLabCITemplate(ctx context.Context, codebase *codebaseApi.Codebase) (string, error) {
	configMapName := GitLabCIDefaultTemplate

	if ann := codebase.GetAnnotations()[codebaseApi.GitLabCITemplateAnnotation]; ann != "" {
		configMapName = ann
	}

	template, err := m.getTemplateFromConfigMap(ctx, configMapName, codebase.Namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get GitLab CI template from ConfigMap %q: %w", configMapName, err)
	}

	return template, nil
}

// getTemplateFromConfigMap retrieves template content from a specific ConfigMap.
func (m *manager) getTemplateFromConfigMap(ctx context.Context, configMapName, namespace string) (string, error) {
	configMap := &corev1.ConfigMap{}

	err := m.client.Get(ctx, client.ObjectKey{
		Name:      configMapName,
		Namespace: namespace,
	}, configMap)
	if err != nil {
		return "", err
	}

	template := configMap.Data[GitLabCIFileName]
	if template == "" {
		return "", fmt.Errorf("no %s template found in ConfigMap %s", GitLabCIFileName, configMapName)
	}

	return template, nil
}
