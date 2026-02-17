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
		name             string
		codebase         *codebaseApi.Codebase
		configMaps       []client.Object
		expectedContains []string
		wantErr          require.ErrorAssertionFunc
	}{
		{
			name: "annotation selects custom ConfigMap",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-go-app",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						codebaseApi.GitLabCITemplateAnnotation: "my-go-template",
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "Go",
					BuildTool: "Go",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-go-template",
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "custom: {{.CodebaseName}}",
					},
				},
			},
			expectedContains: []string{
				"custom: my-go-app",
			},
			wantErr: require.NoError,
		},
		{
			name: "no annotation falls back to default",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "python",
					BuildTool: "pip",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "default: {{.CodebaseName}}",
					},
				},
			},
			expectedContains: []string{
				"default: test-app",
			},
			wantErr: require.NoError,
		},
		{
			name: "annotation pointing to missing ConfigMap returns error",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						codebaseApi.GitLabCITemplateAnnotation: "nonexistent",
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "java",
					BuildTool: "maven",
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), `ConfigMap "nonexistent"`)
			},
		},
		{
			name: "ConfigMap exists but .gitlab-ci.yml key is absent",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
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
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						"some-other-key": "irrelevant",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no .gitlab-ci.yml template found in ConfigMap")
			},
		},
		{
			name: "ConfigMap exists but .gitlab-ci.yml value is empty",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
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
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no .gitlab-ci.yml template found in ConfigMap")
			},
		},
		{
			name: "empty annotation value falls back to default",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						codebaseApi.GitLabCITemplateAnnotation: "",
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "python",
					BuildTool: "pip",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "fallback: {{.CodebaseName}}",
					},
				},
			},
			expectedContains: []string{
				"fallback: test-app",
			},
			wantErr: require.NoError,
		},
		{
			name: "nil annotations map falls back to default",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-app",
					Namespace:   "test-namespace",
					Annotations: nil,
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "java",
					BuildTool: "gradle",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "nil-ann: {{.CodebaseName}}",
					},
				},
			},
			expectedContains: []string{
				"nil-ann: test-app",
			},
			wantErr: require.NoError,
		},
		{
			name: "static template with no placeholders",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
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
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "stages: [build, test]",
					},
				},
			},
			expectedContains: []string{
				"stages: [build, test]",
			},
			wantErr: require.NoError,
		},
		{
			name: "only CodebaseName is substituted",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "Java",
					BuildTool: "Maven",
				},
			},
			configMaps: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GitLabCIDefaultTemplate,
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						".gitlab-ci.yml": "name: {{.CodebaseName}} lang: {{.Lang}} build: {{.BuildTool}}",
					},
				},
			},
			expectedContains: []string{
				"name: my-app",
				"lang: {{.Lang}}",
				"build: {{.BuildTool}}",
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.configMaps...).Build()

			mgr := NewManager(fakeClient)

			err := mgr.InjectGitLabCIConfig(context.Background(), tt.codebase, tmpDir)
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			content, err := os.ReadFile(filepath.Join(tmpDir, GitLabCIFileName))
			require.NoError(t, err)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, string(content), expected)
			}
		})
	}
}

func TestManager_InjectGitLabCIConfig_SkipsIfExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create existing .gitlab-ci.yml
	gitlabCIPath := filepath.Join(tmpDir, GitLabCIFileName)
	existingContent := "existing content"
	require.NoError(t, os.WriteFile(gitlabCIPath, []byte(existingContent), 0644))

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

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	mgr := NewManager(fakeClient)

	err := mgr.InjectGitLabCIConfig(context.Background(), codebase, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(gitlabCIPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}
