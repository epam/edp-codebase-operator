package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestProcessNewVersion_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		codebaseBranch     *codebaseApi.CodebaseBranch
		client             func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client
		wantErr            require.ErrorAssertionFunc
		wantCodebaseBranch func(t *testing.T, cb *codebaseApi.CodebaseBranch)
	}{
		{
			name: "successfully processing new version",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					Version:      util.GetStringP("1.0.0"),
				},
				Status: codebaseApi.CodebaseBranchStatus{
					LastSuccessfulBuild: pointer.String("20"),
					Build:               pointer.String("22"),
				},
			},
			client: func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, codebaseApi.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						cb,
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-codebase",
								Namespace: "default",
							},
							Spec: codebaseApi.CodebaseSpec{
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypeEDP,
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantCodebaseBranch: func(t *testing.T, cb *codebaseApi.CodebaseBranch) {
				require.Equal(t, "0", *cb.Status.Build)
				require.Nil(t, cb.Status.LastSuccessfulBuild)
				require.Equal(t, []string{"1.0.0"}, cb.Status.VersionHistory)
			},
		},
		{
			name: "skip processing new version because of version already exists",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					Version:      util.GetStringP("1.0.0"),
				},
				Status: codebaseApi.CodebaseBranchStatus{
					LastSuccessfulBuild: pointer.String("20"),
					Build:               pointer.String("22"),
					VersionHistory:      []string{"1.0.0"},
				},
			},
			client: func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, codebaseApi.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						cb,
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-codebase",
								Namespace: "default",
							},
							Spec: codebaseApi.CodebaseSpec{
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypeEDP,
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantCodebaseBranch: func(t *testing.T, cb *codebaseApi.CodebaseBranch) {
				require.Equal(t, "22", *cb.Status.Build)
				require.Equal(t, "20", *cb.Status.LastSuccessfulBuild)
				require.Equal(t, []string{"1.0.0"}, cb.Status.VersionHistory)
			},
		},
		{
			name: "skip processing new version because of versioning type is not EDP",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					Version:      util.GetStringP("1.0.0"),
				},
				Status: codebaseApi.CodebaseBranchStatus{
					LastSuccessfulBuild: pointer.String("20"),
					Build:               pointer.String("22"),
					VersionHistory:      []string{"1.0.0"},
				},
			},
			client: func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, codebaseApi.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						cb,
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-codebase",
								Namespace: "default",
							},
							Spec: codebaseApi.CodebaseSpec{
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypDefault,
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantCodebaseBranch: func(t *testing.T, cb *codebaseApi.CodebaseBranch) {
				require.Equal(t, "22", *cb.Status.Build)
				require.Equal(t, "20", *cb.Status.LastSuccessfulBuild)
				require.Equal(t, []string{"1.0.0"}, cb.Status.VersionHistory)
			},
		},
		{
			name: "failed to update CodebaseBranch status",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					Version:      util.GetStringP("1.0.0"),
				},
				Status: codebaseApi.CodebaseBranchStatus{
					LastSuccessfulBuild: pointer.String("20"),
					Build:               pointer.String("22"),
				},
			},
			client: func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, codebaseApi.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-codebase",
								Namespace: "default",
							},
							Spec: codebaseApi.CodebaseSpec{
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypeEDP,
								},
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update CodebaseBranch status")
			},
			wantCodebaseBranch: func(t *testing.T, cb *codebaseApi.CodebaseBranch) {},
		},
		{
			name: "failed - version is not specified",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
				},
				Status: codebaseApi.CodebaseBranchStatus{
					LastSuccessfulBuild: pointer.String("20"),
					Build:               pointer.String("22"),
				},
			},
			client: func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, codebaseApi.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						cb,
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-codebase",
								Namespace: "default",
							},
							Spec: codebaseApi.CodebaseSpec{
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypeEDP,
								},
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to check if branch test-branch has new version")
			},
			wantCodebaseBranch: func(t *testing.T, cb *codebaseApi.CodebaseBranch) {},
		},
		{
			name: "failed - codebase not found",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					Version:      util.GetStringP("1.0.0"),
				},
				Status: codebaseApi.CodebaseBranchStatus{
					LastSuccessfulBuild: pointer.String("20"),
					Build:               pointer.String("22"),
				},
			},
			client: func(t *testing.T, cb *codebaseApi.CodebaseBranch) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, codebaseApi.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get Codebase")
			},
			wantCodebaseBranch: func(t *testing.T, cb *codebaseApi.CodebaseBranch) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ProcessNewVersion{
				Client: tt.client(t, tt.codebaseBranch),
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebaseBranch)
			tt.wantErr(t, err)
			tt.wantCodebaseBranch(t, tt.codebaseBranch)
		})
	}
}
