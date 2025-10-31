package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	gitServerMocks "github.com/epam/edp-codebase-operator/v2/pkg/git/v2/mocks"
)

func TestCheckReferenceExists_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	err := codebaseApi.AddToScheme(scheme)
	require.NoError(t, err)
	err = coreV1.AddToScheme(scheme)
	require.NoError(t, err)
	t.Setenv("WORKING_DIR", t.TempDir())

	tests := []struct {
		name           string
		codebaseBranch *codebaseApi.CodebaseBranch
		objects        []runtime.Object
		gitClient      func() gitproviderv2.Git
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "success, commit hash exists",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "main",
					FromCommit:   "bfba920bd3bdebc9ae1c4475d70391152645b2a4",
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseSpec{
						GitServer: "test-git-server",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-git-server",
						Namespace: "default",
					},
					Spec: codebaseApi.GitServerSpec{
						NameSshKeySecret: "test-ssh-key",
					},
				},
				&coreV1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ssh-key",
						Namespace: "default",
					},
				},
			},
			gitClient: func() gitproviderv2.Git {
				mGit := gitServerMocks.NewMockGit(t)
				mGit.On(
					"Clone",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)
				mGit.On(
					"CheckReference",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)

				return mGit
			},
			wantErr: require.NoError,
		},
		{
			name: "success, branch reference exists",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "feature",
					FromCommit:   "main",
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseSpec{
						GitServer: "test-git-server",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-git-server",
						Namespace: "default",
					},
					Spec: codebaseApi.GitServerSpec{
						NameSshKeySecret: "test-ssh-key",
					},
				},
				&coreV1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ssh-key",
						Namespace: "default",
					},
				},
			},
			gitClient: func() gitproviderv2.Git {
				mGit := gitServerMocks.NewMockGit(t)
				mGit.On(
					"Clone",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)
				mGit.On(
					"CheckReference",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)

				return mGit
			},
			wantErr: require.NoError,
		},
		{
			name: "failed, reference doesn't exist",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "main",
					FromCommit:   "non-existent",
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseSpec{
						GitServer: "test-git-server",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-git-server",
						Namespace: "default",
					},
					Spec: codebaseApi.GitServerSpec{
						NameSshKeySecret: "test-ssh-key",
					},
				},
				&coreV1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ssh-key",
						Namespace: "default",
					},
				},
			},
			gitClient: func() gitproviderv2.Git {
				mGit := gitServerMocks.NewMockGit(t)
				mGit.On(
					"Clone",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)
				mGit.On(
					"CheckReference",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(errors.New("reference not found"))

				return mGit
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "reference non-existent doesn't exist")
			},
		},
		{
			name: "skip, reference is empty",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			gitClient: func() gitproviderv2.Git {
				return gitServerMocks.NewMockGit(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "skip, git branch already created",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "main",
					FromCommit:   "bfba920bd3bdebc9ae1c4475d70391152645b2a4",
				},
				Status: codebaseApi.CodebaseBranchStatus{
					Git: codebaseApi.CodebaseBranchGitStatusBranchCreated,
				},
			},
			gitClient: func() gitproviderv2.Git {
				return gitServerMocks.NewMockGit(t)
			},
			wantErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CheckReferenceExists{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				GitProviderFactory: func(gitServer *codebaseApi.GitServer, secret *coreV1.Secret) gitproviderv2.Git {
					return tt.gitClient()
				},
			}

			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebaseBranch)

			tt.wantErr(t, err)
		})
	}
}
