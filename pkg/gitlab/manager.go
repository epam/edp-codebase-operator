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
func NewManager(client client.Client) Manager {
	return &manager{client: client}
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

	// Simple variable substitution - only codebase name
	content := strings.ReplaceAll(template, "{{.CodebaseName}}", codebase.Name)

	// Write file
	return os.WriteFile(gitlabCIPath, []byte(content), 0644)
}

// getGitLabCITemplate retrieves GitLab CI template with fallback hierarchy.
func (m *manager) getGitLabCITemplate(ctx context.Context, codebase *codebaseApi.Codebase) (string, error) {
	lang := strings.ToLower(codebase.Spec.Lang)
	buildTool := strings.ToLower(codebase.Spec.BuildTool)

	// Try specific language-buildtool combination first
	configMapName := fmt.Sprintf("gitlab-ci-%s-%s", lang, buildTool)
	template, err := m.getTemplateFromConfigMap(ctx, configMapName, codebase.Namespace)

	if err == nil {
		return template, nil
	}

	// Final fallback to default template
	template, err = m.getTemplateFromConfigMap(ctx, GitLabCIDefaultTemplate, codebase.Namespace)
	if err != nil {
		return "", fmt.Errorf("no GitLab CI template found for %s-%s, lang-only, or default", lang, buildTool)
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
