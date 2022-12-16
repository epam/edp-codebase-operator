package chain

import (
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	mockGit "github.com/epam/edp-codebase-operator/v2/controllers/gitserver/mocks"
)

func TestCheckCommitHashExists_ServeRequest(t *testing.T) {
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
		gitClient      func() gitserver.Git
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
			gitClient: func() gitserver.Git {
				mGit := &mockGit.MockGit{}
				mGit.On(
					"CloneRepositoryBySsh",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)
				mGit.On(
					"CommitExists",
					testifymock.Anything,
					testifymock.Anything,
				).Return(true, nil)

				return mGit
			},
			wantErr: require.NoError,
		},
		{
			name: "failed, commit doesn't exist",
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
			gitClient: func() gitserver.Git {
				mGit := &mockGit.MockGit{}
				mGit.On(
					"CloneRepositoryBySsh",
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
					testifymock.Anything,
				).Return(nil)
				mGit.On(
					"CommitExists",
					testifymock.Anything,
					testifymock.Anything,
				).Return(false, nil)

				return mGit
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "commit bfba920bd3bdebc9ae1c4475d70391152645b2a4 doesn't exist")
			},
		},
		{
			name: "skip, commit hash is empty",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			gitClient: func() gitserver.Git {
				return &mockGit.MockGit{}
			},
			wantErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CheckCommitHashExists{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				Git:    tt.gitClient(),
				Log:    logr.Discard(),
			}

			err := c.ServeRequest(tt.codebaseBranch)

			tt.wantErr(t, err)
		})
	}
}
