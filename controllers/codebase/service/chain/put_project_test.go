package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
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
	gerritmocks "github.com/epam/edp-codebase-operator/v2/pkg/gerrit/mocks"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	v2mocks "github.com/epam/edp-codebase-operator/v2/pkg/git/v2/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
	gitprovidermock "github.com/epam/edp-codebase-operator/v2/pkg/gitprovider/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutProject_ServeRequest(t *testing.T) {
	t.Skip("We need to refactor ServeRequest method and rewrite this test accordingly")

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
			NameSshKeySecret: "gerrit-ssh-key",
			GitUser:          "ci",
		},
	}
	gerritGitServerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gerritGitServer.Spec.NameSshKeySecret,
			Namespace: defaultNs,
		},
	}
	githubGitServer := &codebaseApi.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github",
			Namespace: defaultNs,
		},
		Spec: codebaseApi.GitServerSpec{
			GitProvider:      codebaseApi.GitProviderGithub,
			NameSshKeySecret: "github-ssh-key",
		},
	}
	githubGitServerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      githubGitServer.Spec.NameSshKeySecret,
			Namespace: defaultNs,
		},
	}
	gitlabGitServer := &codebaseApi.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab",
			Namespace: defaultNs,
		},
		Spec: codebaseApi.GitServerSpec{
			GitProvider:      codebaseApi.GitProviderGitlab,
			NameSshKeySecret: "gitlab-ssh-key",
		},
	}
	gitlabGitServerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gitlabGitServer.Spec.NameSshKeySecret,
			Namespace: defaultNs,
		},
	}
	defaultGitProvider := func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
		return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
			return gitprovidermock.NewMockGitProjectProvider(t), nil
		}
	}

	tests := []struct {
		name                        string
		codebase                    *codebaseApi.Codebase
		objects                     []client.Object
		gitProviderFactory          func(t *testing.T) gitproviderv2.GitProviderFactory
		gerritClient                func(t *testing.T) gerrit.Client
		gitProvider                 func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error)
		createGitProviderWithConfig func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git
		wantErr                     require.ErrorAssertionFunc
		wantStatus                  func(t *testing.T, status *codebaseApi.CodebaseStatus)
	}{
		{
			name: "gerrit, create strategy - should put project successfully with branch to copy in default branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:                    codebaseApi.Create,
					GitServer:                   gerritGitServer.Name,
					GitUrlPath:                  "/owner/go-repo",
					DefaultBranch:               "master",
					BranchToCopyInDefaultBranch: "main",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				// Create a single mock that will be returned each time the factory is called
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Init", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Commit", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Maybe().
					Return("feature", nil).
					On("RemoveBranch", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("CreateChildBranch", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Checkout", testify.Anything, testify.Anything, testify.Anything, false).
					Maybe().
					Return(nil).
					On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Push", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(nil).
					On(
						"SetHeadToBranch",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
						testify.Anything,
					).
					Return(nil)

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				// Create a single mock that will be returned each time
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Checkout", testify.Anything, testify.Anything, testify.Anything, false).
					Maybe().
					Return(nil).
					On("Init", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Commit", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "gerrit, create strategy - should put empty project successfully",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					DefaultBranch: "master",
					EmptyProject:  true,
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				// Create a single mock that will be returned each time the factory is called
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Maybe().
					Return("master", nil).
					On("Checkout", testify.Anything, testify.Anything, testify.Anything, false).
					Maybe().
					Return(nil).
					On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Push", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(nil).
					On(
						"SetHeadToBranch",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
						testify.Anything,
					).
					Return(nil)

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("Init", testify.Anything, testify.Anything).
						Return(nil).
						On("Commit", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "gerrit, clone strategy - should put project successfully",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				// Create a single mock that will be returned each time the factory is called
				mock := v2mocks.NewMockGit(t)

				mock.
					On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Maybe().
					Return("feature", nil).
					On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
					Maybe().
					Return(nil).
					On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Push", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(nil).
					On(
						"SetHeadToBranch",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
						testify.Anything,
					).
					Return(nil)

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				// Create a single mock that will be returned each time
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "gerrit, clone strategy - failed to set head to branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				// Create a single mock that will be returned each time the factory is called
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Maybe().
					Return("feature", nil).
					On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
					Maybe().
					Return(nil).
					On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Push", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(nil).
					On(
						"SetHeadToBranch",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
						testify.Anything,
					).
					Return(errors.New("failed to set head to branch"))

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to set head to branch")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to set head to branch")
			},
		},
		{
			name: "gerrit, clone strategy - failed to push project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				// Create a single mock that will be returned each time the factory is called
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Maybe().
					Return("feature", nil).
					On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
					Maybe().
					Return(nil).
					On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Push", testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(errors.New("failed to push changes"))

				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(nil)

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				// Create a single mock that will be returned each time
				mock := v2mocks.NewMockGit(t)

				mock.On("CheckPermissions", testify.Anything, testify.Anything).
					Maybe().
					Return(nil).
					On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Maybe().
					Return(nil)

				return func(config gitproviderv2.Config) gitproviderv2.Git {
					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to push changes")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to push changes")
			},
		},
		{
			name: "gerrit, clone strategy - failed create project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(errors.New("failed to create project"))

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to create project")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to create project")
			},
		},
		{
			name: "gerrit, clone strategy - failed to add remote link",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil).
						On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
						Return(errors.New("failed to add remote link"))

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(nil)

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to add remote link")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to add remote link")
			},
		},
		{
			name: "gerrit, clone strategy - failed to create project",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, nil).
					On(
						"CreateProject",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(errors.New("failed to create project"))

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to create project")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to create project")
			},
		},
		{
			name: "gerrit, clone strategy - failed to check project exist",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				mock := gerritmocks.NewMockClient(t)

				mock.
					On(
						"CheckProjectExist",
						testify.Anything,
						testify.Anything,
						testify.Anything,
						gerritGitServer.Spec.GitUser,
						testify.Anything,
						testify.Anything,
					).
					Return(false, errors.New("failed to check project exist"))

				return mock
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to check project exist")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to check project exist")
			},
		},
		{
			name: "gerrit, clone strategy - failed to get GitServer secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					return v2mocks.NewMockGit(t)
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to get GitServer secret")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to get GitServer secret")
			},
		},
		{
			name: "gerrit, clone strategy - failed to checkout branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gerritGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gerritGitServer, gerritGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(errors.New("failed to checkout branch"))

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: defaultGitProvider,
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to checkout branch")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
				assert.Equal(t, util.StatusFailed, status.Status)
				assert.Contains(t, status.DetailedMessage, "failed to checkout branch")
			},
		},
		{
			name: "github, create strategy - should put project successfully",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     githubGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{githubGitServer, githubGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Init", testify.Anything, testify.Anything).
						Return(nil).
						On("Commit", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, false).
						Return(nil).
						On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Push", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := gitprovidermock.NewMockGitProjectProvider(t)

				mock.On("ProjectExists", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(false, nil).
					On("CreateProject", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil).
					On("SetDefaultBranch", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Init", testify.Anything, testify.Anything).
						Return(nil).
						On("Commit", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "github, clone strategy - should put project successfully",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     githubGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{githubGitServer, githubGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil).
						On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Push", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := gitprovidermock.NewMockGitProjectProvider(t)

				mock.On("ProjectExists", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(false, nil).
					On("CreateProject", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil).
					On("SetDefaultBranch", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "gitlab, create strategy - should put project successfully",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Create,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gitlabGitServer, gitlabGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Init", testify.Anything, testify.Anything).
						Return(nil).
						On("Commit", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, false).
						Return(nil).
						On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Push", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := gitprovidermock.NewMockGitProjectProvider(t)

				mock.On("ProjectExists", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(false, nil).
					On("CreateProject", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil).
					On("SetDefaultBranch", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Init", testify.Anything, testify.Anything).
						Return(nil).
						On("Commit", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "gitlab, clone strategy - should put project successfully",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gitlabGitServer, gitlabGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil).
						On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Push", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := gitprovidermock.NewMockGitProjectProvider(t)

				mock.On("ProjectExists", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(false, nil).
					On("CreateProject", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil).
					On("SetDefaultBranch", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil)

				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "gitlab, clone strategy - failed to set default branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy:      codebaseApi.Clone,
					GitServer:     gitlabGitServer.Name,
					GitUrlPath:    "/owner/go-repo",
					Repository:    &codebaseApi.Repository{Url: "https://github.com/owner/repo.git"},
					DefaultBranch: "master",
				},
			},
			objects: []client.Object{gitlabGitServer, gitlabGitServerSecret},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("GetCurrentBranchName", testify.Anything, testify.Anything).
						Return("feature", nil).
						On("Checkout", testify.Anything, testify.Anything, testify.Anything, true).
						Return(nil).
						On("AddRemoteLink", testify.Anything, testify.Anything, testify.Anything).
						Return(nil).
						On("Push", testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return gerritmocks.NewMockClient(t)
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				mock := gitprovidermock.NewMockGitProjectProvider(t)

				mock.On("ProjectExists", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(false, nil).
					On("CreateProject", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(nil).
					On("SetDefaultBranch", testify.Anything, testify.Anything, testify.Anything, testify.Anything, testify.Anything).
					Return(errors.New("failed to set default branch"))

				return func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
					return mock, nil
				}
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return func(config gitproviderv2.Config) gitproviderv2.Git {
					mock := v2mocks.NewMockGit(t)

					mock.On("CheckPermissions", testify.Anything, testify.Anything).
						Return(nil).
						On("Clone", testify.Anything, testify.Anything, testify.Anything, testify.Anything).
						Return(nil)

					return mock
				}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to set default branch")
			},
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.StatusFailed, status.Status)
			},
		},
		{
			name: "should skip import strategy",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Import,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return nil
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return nil
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, "", status.Git)
			},
		},
		{
			name: "should skip if status is already pushed",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Create,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectPushedStatus,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return nil
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return nil
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectPushedStatus, status.Git)
			},
		},
		{
			name: "should skip if status is template already pushed",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "go app",
					Namespace: defaultNs,
				},
				Spec: codebaseApi.CodebaseSpec{
					Strategy: codebaseApi.Create,
				},
				Status: codebaseApi.CodebaseStatus{
					Git: util.ProjectTemplatesPushedStatus,
				},
			},
			gitProviderFactory: func(t *testing.T) gitproviderv2.GitProviderFactory {
				return nil
			},
			gerritClient: func(t *testing.T) gerrit.Client {
				return nil
			},
			gitProvider: func(t *testing.T) func(gitServer *codebaseApi.GitServer, token string) (gitprovider.GitProjectProvider, error) {
				return nil
			},
			createGitProviderWithConfig: func(t *testing.T) func(config gitproviderv2.Config) gitproviderv2.Git {
				return nil
			},
			wantErr: require.NoError,
			wantStatus: func(t *testing.T, status *codebaseApi.CodebaseStatus) {
				assert.Equal(t, util.ProjectTemplatesPushedStatus, status.Git)
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
				tt.gitProvider(t),
				tt.gitProviderFactory(t),
				tt.createGitProviderWithConfig(t),
			)

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebase)

			tt.wantErr(t, err)

			if err != nil {
				tt.wantStatus(t, &tt.codebase.Status)
				return
			}

			processedCodebase := &codebaseApi.Codebase{}
			if err = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      tt.codebase.Name,
					Namespace: tt.codebase.Namespace,
				},
				processedCodebase,
			); err != nil {
				require.NoError(t, err)
			}

			tt.wantStatus(t, &processedCodebase.Status)
		})
	}
}
