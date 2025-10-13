package gitlab_ci

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestManager_InjectGitLabCIConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		codebase       *codebaseApi.Codebase
		configMaps     []client.Object
		expectedInFile string
	}{
		{
			name: "java maven project with specific ConfigMap",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-java-app",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "java",
					BuildTool: "maven",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-ci-java-maven",
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "variables:\n  CODEBASE_NAME: \"{{.CodebaseName}}\"\ninclude:\n  - component: $CI_SERVER_FQDN/kuberocketci/ci-java17-mvn/build@0.1.1",
					},
				},
			},
			expectedInFile: "CODEBASE_NAME: \"test-java-app\"",
		},
		{
			name: "go project with specific template",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-go-app",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "go",
					BuildTool: "go",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-ci-go-go",
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "variables:\n  CODEBASE_NAME: \"{{.CodebaseName}}\"\ninclude:\n  - component: $CI_SERVER_FQDN/kuberocketci/ci-golang/build@0.1.1",
					},
				},
			},
			expectedInFile: "CODEBASE_NAME: \"test-go-app\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "gitlab-ci-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create fake Kubernetes client with ConfigMaps
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.configMaps...).Build()

			manager := NewManager(fakeClient)
			ctx := context.Background()

			// Test injection
			err = manager.InjectGitLabCIConfig(ctx, tt.codebase, tmpDir)
			require.NoError(t, err)

			// Verify file was created
			gitlabCIPath := filepath.Join(tmpDir, GitLabCIFileName)
			content, err := os.ReadFile(gitlabCIPath)
			require.NoError(t, err)

			// Verify content contains expected substitutions
			contentStr := string(content)
			assert.Contains(t, contentStr, tt.expectedInFile)
		})
	}
}

func TestManager_InjectGitLabCIConfig_SkipsIfExists(t *testing.T) {
	t.Parallel()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "gitlab-ci-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create existing .gitlab-ci.yml
	gitlabCIPath := filepath.Join(tmpDir, GitLabCIFileName)
	existingContent := "existing content"
	err = os.WriteFile(gitlabCIPath, []byte(existingContent), 0644)
	require.NoError(t, err)

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Lang:      "java",
			BuildTool: "maven",
		},
	}

	// Create fake client (ConfigMaps not needed for this test)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	// Test injection
	err = manager.InjectGitLabCIConfig(ctx, codebase, tmpDir)
	require.NoError(t, err)

	// Verify file was not overwritten
	content, err := os.ReadFile(gitlabCIPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}

func TestManager_ConfigMapFallbackHierarchy(t *testing.T) {
	t.Parallel()

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Lang:      "python",
			BuildTool: "pip",
		},
	}

	// Create ConfigMaps: only default (no specific template)
	configMaps := []client.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gitlab-ci-default",
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				".gitlab-ci.yml": "default-fallback: {{.CodebaseName}}",
			},
		},
	}

	// Create fake client
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMaps...).Build()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "gitlab-ci-fallback-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager := NewManager(fakeClient)
	ctx := context.Background()

	// Test injection - should use default fallback since no specific template exists
	err = manager.InjectGitLabCIConfig(ctx, codebase, tmpDir)
	require.NoError(t, err)

	// Verify file was created with default fallback content
	gitlabCIPath := filepath.Join(tmpDir, GitLabCIFileName)
	content, err := os.ReadFile(gitlabCIPath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "default-fallback: test-app")
}
