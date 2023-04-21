package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	edpComponentApi "github.com/epam/edp-component-operator/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutGitWebRepoUrl_ServeRequest(t *testing.T) {
	t.Parallel()

	schema := runtime.NewScheme()

	require.NoError(t, codebaseApi.AddToScheme(schema))
	require.NoError(t, edpComponentApi.AddToScheme(schema))

	const namespace = "test-ns"

	gitURL := "/test-owner/test-repo"

	tests := []struct {
		name       string
		codebase   *codebaseApi.Codebase
		gitServer  *codebaseApi.GitServer
		k8sObjects []client.Object
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name: "should put GitWebUrl in codebase status",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace, ResourceVersion: "1"},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
				Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGithub, GitHost: "github.com"},
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGithub, GitHost: "github.com"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace, ResourceVersion: "1"},
					Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
					Status:     codebaseApi.CodebaseStatus{},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "should fail if git server is not found",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "unknown-git-server", GitUrlPath: &gitURL},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer:  nil,
			k8sObjects: []client.Object{},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "failed to get git server unknown-git-server")
			},
		},
		{
			name: "should fail if gitUrlPath is not set",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server"},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer: &codebaseApi.GitServer{ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace}},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGithub, GitHost: "github.com"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "failed to get GitUrlPath for codebase test, git url path is empty")
			},
		},
		{
			name: "should fail if Provider is not supported",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer: &codebaseApi.GitServer{ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace}},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: "bitbucket"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
					Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "unsupported Git provider bitbucket")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(schema).WithObjects(tt.k8sObjects...).Build()
			s := NewPutGitWebRepoUrl(k8sClient)

			gotErr := s.ServeRequest(context.Background(), tt.codebase)
			tt.wantErr(t, gotErr)
		})
	}
}

func TestPutGitWebRepoUrl_getGitWebURL(t *testing.T) {
	t.Parallel()

	schema := runtime.NewScheme()

	require.NoError(t, codebaseApi.AddToScheme(schema))
	require.NoError(t, edpComponentApi.AddToScheme(schema))

	const namespace = "test-ns"

	gitURL := "/test-owner/test-repo"

	tests := []struct {
		name           string
		codebase       *codebaseApi.Codebase
		gitServer      *codebaseApi.GitServer
		k8sObjects     []client.Object
		wantErr        require.ErrorAssertionFunc
		expectedWebUrl string
	}{
		{
			name:           "should return GitWebUrl for Github",
			expectedWebUrl: "https://github.com/test-owner/test-repo",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace, ResourceVersion: "1"},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
				Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGithub, GitHost: "github.com"},
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGithub, GitHost: "github.com"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace, ResourceVersion: "1"},
					Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
					Status:     codebaseApi.CodebaseStatus{},
				},
			},
			wantErr: require.NoError,
		},
		{
			name:           "should return correct GitWebUrl for Gerrit with trailing slash in EDPComponent.Url",
			expectedWebUrl: "https://gerrit.example.com/gitweb?p=test-app.git",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace, ResourceVersion: "1"},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: pointer.String("/test-app")},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
				Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGerrit, GitHost: "gerrit"},
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGerrit, GitHost: "gerrit"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace, ResourceVersion: "1"},
				},
				&edpComponentApi.EDPComponent{
					ObjectMeta: metaV1.ObjectMeta{Name: "gerrit", Namespace: namespace},
					Spec:       edpComponentApi.EDPComponentSpec{Url: "https://gerrit.example.com/"},
				},
			},
			wantErr: require.NoError,
		},
		{
			name:           "should fail if gerrit EDP Component is not found",
			expectedWebUrl: "",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
				Status:     codebaseApi.CodebaseStatus{},
			},
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
				Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGerrit, GitHost: "gerrit"},
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: codebaseApi.GitProviderGerrit, GitHost: "github.com"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "failed to fetch EDPComponent gerrit")
			},
		},
		{
			name:           "should fail with UnsupportedGitProvider error if GitProvider is not supported",
			expectedWebUrl: "",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec:       codebaseApi.CodebaseSpec{GitServer: "git-server", GitUrlPath: &gitURL},
			},
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
				Spec:       codebaseApi.GitServerSpec{GitProvider: "bitbucket", GitHost: "bitbucket"},
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{Name: "git-server", Namespace: namespace},
					Spec:       codebaseApi.GitServerSpec{GitProvider: "bitbucket", GitHost: "bitbucket"},
				},
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{Name: "test", Namespace: namespace},
				},
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "unsupported Git provider")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(schema).WithObjects(tt.k8sObjects...).Build()
			s := NewPutGitWebRepoUrl(k8sClient)

			webUrl, gotErr := s.getGitWebURL(context.Background(), tt.gitServer, tt.codebase)
			assert.Equal(t, tt.expectedWebUrl, webUrl)
			tt.wantErr(t, gotErr)
		})
	}
}
