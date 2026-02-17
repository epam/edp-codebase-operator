package chain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	testify "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitmocks "github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
	gitlabci "github.com/epam/edp-codebase-operator/v2/pkg/gitlab"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const testGitLabCIConfigData = "variables:\n  CODEBASE_NAME: \"{{.CodebaseName}}\"\ninclude:\n  " +
	"- component: $CI_SERVER_FQDN/kuberocketci/ci-java17-mvn/build@0.1.1"

func TestPutGitLabCIConfig_ServeRequest(t *testing.T) {
	const defaultNs = "default"

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	gitlabGitServer := &codebaseApi.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab",
			Namespace: defaultNs,
		},
		Spec: codebaseApi.GitServerSpec{
			GitProvider:      codebaseApi.GitProviderGitlab,
			GitHost:          "gitlab.com",
			GitUser:          "git",
			SshPort:          22,
			NameSshKeySecret: "gitlab-secret",
		},
	}

	gitlabGitServerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab-secret",
			Namespace: defaultNs,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName:         []byte("fake-ssh-key"),
			util.GitServerSecretTokenField: []byte("fake-token"),
		},
	}

	tests := []struct {
		name       string
		codebase   *codebaseApi.Codebase
		objects    []client.Object
		gitClient  func(t *testing.T) gitproviderv2.Git
		setup      func(t *testing.T, wd string)
		wantErr    require.ErrorAssertionFunc
		wantStatus func(t *testing.T, codebase *codebaseApi.Codebase)
	}{
		{
			name: "skip when not GitLab CI",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					CiTool: "jenkins",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				return gitmocks.NewMockGit(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "skip when file already exists",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					CiTool: util.CIGitLab,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				return gitmocks.NewMockGit(t)
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
				gitlabCIPath := filepath.Join(wd, gitlabci.GitLabCIFileName)
				require.NoError(t, os.WriteFile(gitlabCIPath, []byte("existing config"), 0644))
			},
			wantErr: require.NoError,
		},
		{
			name: "successfully inject GitLab CI config",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
					Lang:          "java",
					BuildTool:     "maven",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{
				gitlabGitServer,
				gitlabGitServerSecret,
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      gitlabci.GitLabCIDefaultTemplate,
						Namespace: defaultNs,
					},
					Data: map[string]string{
						".gitlab-ci.yml": testGitLabCIConfigData,
					},
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)

				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(nil)
				mock.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("master", nil)
				mock.On("Commit", testify.Anything, testify.Anything, "Add GitLab CI configuration").
					Return(nil)
				mock.On("Push", testify.Anything, testify.Anything, gitproviderv2.RefSpecPushAllBranches).
					Return(nil)

				return mock
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)
			},
		},
		{
			name: "successfully inject GitLab CI config with annotation",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
					Annotations: map[string]string{
						codebaseApi.GitLabCITemplateAnnotation: "my-custom-template",
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
					Lang:          "java",
					BuildTool:     "maven",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{
				gitlabGitServer,
				gitlabGitServerSecret,
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-custom-template",
						Namespace: defaultNs,
					},
					Data: map[string]string{
						".gitlab-ci.yml": "custom: {{.CodebaseName}}",
					},
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)

				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(nil)
				mock.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("master", nil)
				mock.On("Commit", testify.Anything, testify.Anything, "Add GitLab CI configuration").
					Return(nil)
				mock.On("Push", testify.Anything, testify.Anything, gitproviderv2.RefSpecPushAllBranches).
					Return(nil)

				return mock
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)
			},
		},
		{
			name: "skip when status is gitlab_ci_pushed",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					CiTool: util.CIGitLab,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectGitLabCIPushedStatus,
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				return gitmocks.NewMockGit(t)
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)
			},
		},
		{
			name: "skip when status is templates_pushed",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					CiTool: util.CIGitLab,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectTemplatesPushedStatus,
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				return gitmocks.NewMockGit(t)
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.ProjectTemplatesPushedStatus, codebase.Status.Git)
			},
		},
		{
			name: "failed to get GitServer",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     "non-existent-server",
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				return gitmocks.NewMockGit(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get GitServer")
			},
		},
		{
			name: "failed to get GitServer secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{gitlabGitServer},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				return gitmocks.NewMockGit(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get GitServer secret")
			},
		},
		{
			name: "failed to clone repository",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{gitlabGitServer, gitlabGitServerSecret},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)
				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(errors.New("failed to clone git repository"))

				return mock
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to clone git repository")
			},
		},
		{
			name: "failed to checkout default branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{gitlabGitServer, gitlabGitServerSecret},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)
				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(nil)
				mock.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("", errors.New("failed to get current branch"))

				return mock
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get current branch")
			},
		},
		{
			name: "failed to inject GitLab CI config - missing ConfigMap",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
					Lang:          "java",
					BuildTool:     "maven",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{gitlabGitServer, gitlabGitServerSecret},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)
				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(nil)
				mock.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("master", nil)

				return mock
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to inject GitLab CI config")
			},
		},
		{
			name: "failed to commit changes",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
					Lang:          "java",
					BuildTool:     "maven",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{
				gitlabGitServer,
				gitlabGitServerSecret,
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      gitlabci.GitLabCIDefaultTemplate,
						Namespace: defaultNs,
					},
					Data: map[string]string{
						".gitlab-ci.yml": testGitLabCIConfigData,
					},
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)
				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(nil)
				mock.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("master", nil)
				mock.On("Commit", testify.Anything, testify.Anything, "Add GitLab CI configuration").
					Return(errors.New("failed to commit changes"))

				return mock
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to commit changes")
			},
		},
		{
			name: "failed to push changes",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "java-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					CiTool:        util.CIGitLab,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/java-repo",
					Repository:    &codebaseApi.Repository{Url: "https://gitlab.com/owner/java-repo.git"},
					DefaultBranch: "master",
					Lang:          "java",
					BuildTool:     "maven",
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			objects: []client.Object{
				gitlabGitServer,
				gitlabGitServerSecret,
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      gitlabci.GitLabCIDefaultTemplate,
						Namespace: defaultNs,
					},
					Data: map[string]string{
						".gitlab-ci.yml": testGitLabCIConfigData,
					},
				},
			},
			gitClient: func(t *testing.T) gitproviderv2.Git {
				mock := gitmocks.NewMockGit(t)
				mock.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(nil)
				mock.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("master", nil)
				mock.On("Commit", testify.Anything, testify.Anything, "Add GitLab CI configuration").
					Return(nil)
				mock.On("Push", testify.Anything, testify.Anything, gitproviderv2.RefSpecPushAllBranches).
					Return(errors.New("failed to push changes"))

				return mock
			},
			setup: func(t *testing.T, wd string) {
				require.NoError(t, os.MkdirAll(wd, 0755))
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to push changes")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use t.TempDir() for automatic cleanup
			tmpDir := t.TempDir()

			// Set the working directory environment variable
			t.Setenv(util.WorkDirEnv, tmpDir)

			// Run test-specific setup if provided
			if tt.setup != nil {
				wd := util.GetWorkDir(tt.codebase.Name, tt.codebase.Namespace)
				tt.setup(t, wd)
			}

			// Prepare all objects including the codebase
			allObjects := append(tt.objects, tt.codebase)

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(allObjects...).
				WithStatusSubresource(&codebaseApi.Codebase{}).
				Build()

			// Create git client once and reuse
			gitClient := tt.gitClient(t)

			h := NewPutGitLabCIConfig(
				k8sClient,
				gitlabci.NewManager(k8sClient),
				func(gitproviderv2.Config) gitproviderv2.Git {
					return gitClient
				},
			)

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebase)
			tt.wantErr(t, err)

			if tt.wantStatus != nil {
				tt.wantStatus(t, tt.codebase)
			}
		})
	}
}
