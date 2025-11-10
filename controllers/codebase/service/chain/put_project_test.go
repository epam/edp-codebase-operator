package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	testify "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	gerritMocks "github.com/epam/edp-codebase-operator/v2/pkg/gerrit/mocks"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitmocks "github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutProject_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	const defaultNs = "default"

	tests := []struct {
		name                    string
		codebase                *codebaseApi.Codebase
		objects                 []client.Object
		gitProviderFactory      func(t *testing.T) gitproviderv2.GitProviderFactory
		gitProvider             func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error)
		wantErr                 require.ErrorAssertionFunc
		wantStatus              func(t *testing.T, status codebaseApi.CodebaseStatus)
		wantCodebaseErrorStatus func(t *testing.T, codebase *codebaseApi.Codebase)
	}{
		{
			name: "skip for Import strategy",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Import,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return nil
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Empty(t, status.Git)
			},
		},
		{
			name: "skip when already pushed with ProjectPushedStatus",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Clone,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return nil
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "skip when already pushed with ProjectGitLabCIPushedStatus",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Create,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectGitLabCIPushedStatus,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return nil
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectGitLabCIPushedStatus, status.Git)
			},
		},
		{
			name: "skip when already pushed with ProjectTemplatesPushedStatus",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Clone,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectTemplatesPushedStatus,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return nil
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectTemplatesPushedStatus, status.Git)
			},
		},
		{
			name: "successfully create empty project with third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "gitlab",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGitlab,
						GitHost:          "gitlab.example.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "gitlab-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, nil)
				mock.EXPECT().CreateProject(testify.Anything, testify.Anything, testify.Anything, "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetDefaultBranch(testify.Anything, testify.Anything, testify.Anything, "test-app", "main").
					Return(nil)
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "successfully create empty project if project already exists in third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "github",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGithub,
						GitHost:          "github.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "github-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(true, nil)
				mock.EXPECT().SetDefaultBranch(testify.Anything, testify.Anything, testify.Anything, "test-app", "main").
					Return(nil)
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "successfully create non-empty project with third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "gitlab",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  false,
					Lang:          "go",
					BuildTool:     "go",
					Framework:     "beego",
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGitlab,
						GitHost:          "gitlab.example.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "gitlab-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Clone(testify.Anything, "https://github.com/epmd-edp/go-go-beego.git", testify.Anything).Return(nil)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, nil)
				mock.EXPECT().CreateProject(testify.Anything, testify.Anything, testify.Anything, "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetDefaultBranch(testify.Anything, testify.Anything, testify.Anything, "test-app", "main").
					Return(nil)
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "failed to check if project exists in third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "gitlab",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGitlab,
						GitHost:          "gitlab.example.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "gitlab-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, errors.New("API connection failed"))
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to check if project exists")
			},
		},
		{
			name: "failed to create project in third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "github",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGithub,
						GitHost:          "github.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "github-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, nil)
				mock.EXPECT().CreateProject(testify.Anything, testify.Anything, testify.Anything, "test-app", testify.Anything).
					Return(errors.New("permission denied"))
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to create project")
			},
		},
		{
			name: "failed to push to third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "gitlab",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGitlab,
						GitHost:          "gitlab.example.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "gitlab-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(errors.New("push rejected"))
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, nil)
				mock.EXPECT().CreateProject(testify.Anything, testify.Anything, testify.Anything, "test-app", testify.Anything).
					Return(nil)
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to push changes and tags into git")
			},
		},
		{
			name: "failed to set default branch in third-party git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "github",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGithub,
						GitHost:          "github.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "github-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, nil)
				mock.EXPECT().CreateProject(testify.Anything, testify.Anything, testify.Anything, "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetDefaultBranch(testify.Anything, testify.Anything, testify.Anything, "test-app", "main").
					Return(errors.New("branch does not exist"))
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to set default branch")
			},
		},
		{
			name: "successfully create project when SetDefaultBranch is not supported by git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "gitlab",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGitlab,
						GitHost:          "gitlab.example.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "gitlab-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := mocks.NewMockGitProjectProvider(t)
				mock.EXPECT().ProjectExists(testify.Anything, testify.Anything, testify.Anything, "test-app").
					Return(false, nil)
				mock.EXPECT().CreateProject(testify.Anything, testify.Anything, testify.Anything, "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetDefaultBranch(testify.Anything, testify.Anything, testify.Anything, "test-app", "main").
					Return(gitprovider.ErrApiNotSupported)
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "failed to clone template repository for non-empty project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "github",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  false,
					Lang:          "go",
					BuildTool:     "go",
					Framework:     "beego",
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGithub,
						GitHost:          "github.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "github-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Clone(testify.Anything, "https://github.com/epmd-edp/go-go-beego.git", testify.Anything).
					Return(errors.New("repository not found"))
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mocks.NewMockGitProjectProvider(t), nil
				}
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to clone template project")
			},
		},
		{
			name: "failed to init empty repository",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     "gitlab",
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.GitServerSpec{
						GitProvider:      codebaseApi.GitProviderGitlab,
						GitHost:          "gitlab.example.com",
						GitUser:          "edp-ci",
						NameSshKeySecret: "gitlab-access-token",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitlab-access-token",
						Namespace: defaultNs,
					},
					Data: map[string][]byte{
						"token": []byte("fake-token"),
					},
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(errors.New("failed to initialize git repository"))
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mocks.NewMockGitProjectProvider(t), nil
				}
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to create empty git repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(util.WorkDirEnv, t.TempDir())

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.codebase).
				WithObjects(tt.objects...).
				WithStatusSubresource(tt.codebase).
				WithStatusSubresource(tt.objects...).
				Build()

			h := NewPutProject(
				k8sClient,
				gerritMocks.NewMockClient(t),
				tt.gitProvider(t),
				tt.gitProviderFactory(t),
			)

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebase)

			tt.wantErr(t, err)

			processedCodebase := &codebaseApi.Codebase{}

			require.NoError(t,
				k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      tt.codebase.Name,
						Namespace: tt.codebase.Namespace,
					},
					processedCodebase,
				),
			)

			if tt.wantCodebaseErrorStatus != nil {
				tt.wantCodebaseErrorStatus(t, tt.codebase)
			}

			if tt.wantStatus != nil {
				tt.wantStatus(t, processedCodebase.Status)
			}
		})
	}
}

func TestPutProject_ServeRequest_Gerrit(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	defaultNs := "default"
	gerritGitServer := &codebaseApi.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit",
			Namespace: defaultNs,
		},
		Spec: codebaseApi.GitServerSpec{
			GitProvider:      codebaseApi.GitProviderGerrit,
			GitHost:          "gerrit.example.com",
			GitUser:          "ci",
			SshPort:          29418,
			NameSshKeySecret: "gerrit-ssh-key",
		},
	}
	gerritGitServerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerritGitServer.Spec.NameSshKeySecret,
			Namespace: defaultNs,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake-ssh-key"),
		},
	}

	tests := []struct {
		name                    string
		codebase                *codebaseApi.Codebase
		objects                 []client.Object
		gitProviderFactory      func(t *testing.T) gitproviderv2.GitProviderFactory
		gerritClient            func(t *testing.T) gerrit.Client
		wantErr                 require.ErrorAssertionFunc
		wantStatus              func(t *testing.T, status codebaseApi.CodebaseStatus)
		wantCodebaseErrorStatus func(t *testing.T, codebase *codebaseApi.Codebase)
	}{
		{
			name: "successfully create empty Gerrit project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(false, nil)
				mock.EXPECT().CreateProject(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetHeadToBranch(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", "main", testify.Anything).
					Return(nil)
				return mock
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "successfully create empty Gerrit project if project already exists",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(true, nil)
				mock.EXPECT().SetHeadToBranch(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", "main", testify.Anything).
					Return(nil)
				return mock
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "successfully create non-empty Gerrit project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  false,
					Lang:          "go",
					BuildTool:     "go",
					Framework:     "beego",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Clone(testify.Anything, "https://github.com/epmd-edp/go-go-beego.git", testify.Anything).Return(nil)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(false, nil)
				mock.EXPECT().CreateProject(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetHeadToBranch(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", "main", testify.Anything).
					Return(nil)
				return mock
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status codebaseApi.CodebaseStatus) {
				require.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "failed to check if Gerrit project exists",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(false, errors.New("SSH connection failed"))
				return mock
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to check if project exist in Gerrit")
			},
		},
		{
			name: "failed to create Gerrit project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(false, nil)
				mock.EXPECT().CreateProject(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(errors.New("permission denied"))
				return mock
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to create gerrit project")
			},
		},
		{
			name: "failed to push to Gerrit",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(errors.New("push rejected"))
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(false, nil)
				mock.EXPECT().CreateProject(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(nil)
				return mock
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to push changes and tags into git")
			},
		},
		{
			name: "failed to set HEAD to branch in Gerrit",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Commit(testify.Anything, testify.Anything, "Initial commit", testify.Anything).Return(nil)
				mock.EXPECT().GetCurrentBranchName(testify.Anything, testify.Anything).Return("main", nil)
				mock.EXPECT().AddRemoteLink(testify.Anything, testify.Anything, testify.Anything).Return(nil)
				mock.EXPECT().Push(testify.Anything, testify.Anything, testify.Anything, testify.Anything).Return(nil)
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritMocks.NewMockClient(t)
				mock.EXPECT().CheckProjectExist(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(false, nil)
				mock.EXPECT().CreateProject(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", testify.Anything).
					Return(nil)
				mock.EXPECT().SetHeadToBranch(int32(29418), "fake-ssh-key", "gerrit.example.com", "ci", "test-app", "main", testify.Anything).
					Return(errors.New("branch does not exist"))
				return mock
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "set remote HEAD for codebase test-app to default branch main has been failed")
			},
		},
		{
			name: "failed to clone template repository for non-empty project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  false,
					Lang:          "go",
					BuildTool:     "go",
					Framework:     "beego",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Clone(testify.Anything, "https://github.com/epmd-edp/go-go-beego.git", testify.Anything).
					Return(errors.New("repository not found"))
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritMocks.NewMockClient(t)
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to clone template project")
			},
		},
		{
			name: "failed to init empty repository",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/test-app",
					DefaultBranch: "main",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				mock := gitmocks.NewMockGit(t)
				mock.EXPECT().Init(testify.Anything, testify.Anything).Return(errors.New("failed to initialize git repository"))
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritMocks.NewMockClient(t)
			},
			wantErr: require.Error,
			wantCodebaseErrorStatus: func(t *testing.T, codebase *codebaseApi.Codebase) {
				require.Equal(t, util.StatusFailed, codebase.Status.Status)
				require.Contains(t, codebase.Status.DetailedMessage, "failed to create empty git repository")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(util.WorkDirEnv, t.TempDir())

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.codebase).
				WithObjects(tt.objects...).
				WithStatusSubresource(tt.codebase).
				WithStatusSubresource(tt.objects...).
				Build()

			h := NewPutProject(
				k8sClient,
				tt.gerritClient(t),
				func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mocks.NewMockGitProjectProvider(t), nil
				},
				tt.gitProviderFactory(t),
			)

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebase)

			tt.wantErr(t, err)

			if tt.wantCodebaseErrorStatus != nil {
				tt.wantCodebaseErrorStatus(t, tt.codebase)
			}

			processedCodebase := &codebaseApi.Codebase{}

			require.NoError(t,
				k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      tt.codebase.Name,
						Namespace: tt.codebase.Namespace,
					},
					processedCodebase,
				),
			)

			if tt.wantStatus != nil {
				tt.wantStatus(t, processedCodebase.Status)
			}
		})
	}
}
