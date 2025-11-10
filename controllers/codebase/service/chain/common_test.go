package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	testify "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitMocks "github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestUpdateGitStatusWithPatch_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: "",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)

	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)

	// Verify status was actually patched in the cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, updated.Status.Git)
}

func TestUpdateGitStatusWithPatch_Idempotent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: util.ProjectPushedStatus,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)

	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)
}

func TestUpdateGitStatusWithPatch_StatusTransition(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: util.ProjectPushedStatus,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	// Execute - transition from ProjectPushedStatus to ProjectGitLabCIPushedStatus
	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectGitLabCIPushedStatus,
	)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)

	// Verify in cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectGitLabCIPushedStatus, updated.Status.Git)
}

func TestUpdateGitStatusWithPatch_PreservesOtherStatusFields(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git:        "",
			WebHookRef: "webhook-123",
			GitWebUrl:  "https://git.example.com/repo",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	// Execute
	err := updateGitStatusWithPatch(
		context.Background(),
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)

	// Verify - only Git field changed, others preserved
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)
	assert.Equal(t, "webhook-123", codebase.Status.WebHookRef)
	assert.Equal(t, "https://git.example.com/repo", codebase.Status.GitWebUrl)

	// Verify in cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, updated.Status.Git)
	assert.Equal(t, "webhook-123", updated.Status.WebHookRef)
	assert.Equal(t, "https://git.example.com/repo", updated.Status.GitWebUrl)
}

func TestUpdateGitStatusWithPatch_SequentialUpdates(t *testing.T) {
	// Setup - This test simulates the chain handler scenario
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	codebase := &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: "",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(codebase).
		WithStatusSubresource(codebase).
		Build()

	ctx := context.Background()

	// Simulate sequential updates as in chain handlers
	// 1. PutProject sets ProjectPushedStatus
	err := updateGitStatusWithPatch(
		ctx,
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectPushedStatus,
	)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectPushedStatus, codebase.Status.Git)

	// 2. PutGitLabCIConfig sets ProjectGitLabCIPushedStatus
	err = updateGitStatusWithPatch(
		ctx,
		fakeClient,
		codebase,
		codebaseApi.RepositoryProvisioning,
		util.ProjectGitLabCIPushedStatus,
	)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectGitLabCIPushedStatus, codebase.Status.Git)

	// 3. PutDeployConfigs sets ProjectTemplatesPushedStatus
	err = updateGitStatusWithPatch(
		ctx,
		fakeClient,
		codebase,
		codebaseApi.SetupDeploymentTemplates,
		util.ProjectTemplatesPushedStatus,
	)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectTemplatesPushedStatus, codebase.Status.Git)

	// Verify final state in cluster
	updated := &codebaseApi.Codebase{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      fakeName,
		Namespace: fakeNamespace,
	}, updated)
	assert.NoError(t, err)
	assert.Equal(t, util.ProjectTemplatesPushedStatus, updated.Status.Git)
}

func TestPrepareGitRepository(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	gitServer := &codebaseApi.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gitserver",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "github.com",
			GitUser:          "git",
			NameSshKeySecret: "git-secret",
			SshPort:          22,
			GitProvider:      codebaseApi.GitProviderGithub,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-secret",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("test-ssh-key"),
		},
	}

	tests := []struct {
		name      string
		codebase  *codebaseApi.Codebase
		objects   []client.Object
		gitClient func(t *testing.T) *gitMocks.MockGit
		setup     func(t *testing.T)
		wantErr   require.ErrorAssertionFunc
		want      func(t *testing.T, gitCtx *GitRepositoryContext)
	}{
		{
			name: "successfully prepare repository - clone required",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:     gitServer.Name,
					DefaultBranch: "main",
					Strategy:      codebaseApi.Create,
					GitUrlPath:    fakeName,
				},
			},
			objects: []client.Object{gitServer, secret},
			gitClient: func(t *testing.T) *gitMocks.MockGit {
				m := gitMocks.NewMockGit(t)
				m.On("Clone", testify.Anything, testify.Anything, testify.Anything).Return(nil)
				m.On("GetCurrentBranchName", testify.Anything, testify.Anything).Return("main", nil)
				return m
			},
			wantErr: require.NoError,
			want: func(t *testing.T, gitCtx *GitRepositoryContext) {
				require.NotNil(t, gitCtx)
				assert.Equal(t, "test-ssh-key", gitCtx.PrivateSSHKey)
				assert.Equal(t, gitServer.Name, gitCtx.GitServer.Name)
				assert.Equal(t, secret.Name, gitCtx.GitServerSecret.Name)
				assert.NotEmpty(t, gitCtx.WorkDir)
			},
		},
		{
			name: "failed to get GitServer",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:     "missing-gitserver",
					DefaultBranch: "main",
				},
			},
			gitClient: func(t *testing.T) *gitMocks.MockGit {
				return gitMocks.NewMockGit(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get GitServer")
			},
			want: func(t *testing.T, gitCtx *GitRepositoryContext) {
				assert.Nil(t, gitCtx)
			},
		},
		{
			name: "failed to get GitServer secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:     gitServer.Name,
					DefaultBranch: "main",
				},
			},
			objects: []client.Object{gitServer},
			gitClient: func(t *testing.T) *gitMocks.MockGit {
				return gitMocks.NewMockGit(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get GitServer secret")
			},
			want: func(t *testing.T, gitCtx *GitRepositoryContext) {
				assert.Nil(t, gitCtx)
			},
		},
		{
			name: "failed to clone repository",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:     gitServer.Name,
					DefaultBranch: "main",
					Strategy:      codebaseApi.Create,
				},
			},
			objects: []client.Object{gitServer, secret},
			gitClient: func(t *testing.T) *gitMocks.MockGit {
				m := gitMocks.NewMockGit(t)
				m.On("Clone", testify.Anything, testify.Anything, testify.Anything).
					Return(assert.AnError)
				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to clone git repository")
			},
			want: func(t *testing.T, gitCtx *GitRepositoryContext) {
				assert.Nil(t, gitCtx)
			},
		},
		{
			name: "failed to checkout default branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:     gitServer.Name,
					DefaultBranch: "main",
					Strategy:      codebaseApi.Create,
					GitUrlPath:    fakeName,
					Repository:    &codebaseApi.Repository{Url: "https://github.com/test/repo.git"},
				},
			},
			objects: []client.Object{
				gitServer,
				secret,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "repository-codebase-fake-name-temp",
						Namespace: fakeNamespace,
					},
					Data: map[string][]byte{
						"username": []byte("user"),
						"password": []byte("pass"),
					},
				},
			},
			gitClient: func(t *testing.T) *gitMocks.MockGit {
				m := gitMocks.NewMockGit(t)
				m.On("Clone", testify.Anything, testify.Anything, testify.Anything).Return(nil)
				m.On("GetCurrentBranchName", testify.Anything, testify.Anything).
					Return("", errors.New("failed to get current branch"))
				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get current branch")
			},
			want: func(t *testing.T, gitCtx *GitRepositoryContext) {
				assert.Nil(t, gitCtx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(util.WorkDirEnv, tmpDir)

			if tt.setup != nil {
				tt.setup(t)
			}

			allObjects := append(tt.objects, tt.codebase)

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(allObjects...).
				Build()

			gitProvider := tt.gitClient(t)

			gitCtx, err := PrepareGitRepository(
				context.Background(),
				k8sClient,
				tt.codebase,
				func(cfg gitproviderv2.Config) gitproviderv2.Git {
					return gitProvider
				},
			)

			tt.wantErr(t, err)

			if tt.want != nil {
				tt.want(t, gitCtx)
			}
		})
	}
}
