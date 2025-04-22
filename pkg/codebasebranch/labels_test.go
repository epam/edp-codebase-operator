package codebasebranch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestAddCodebaseLabel(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	type args struct {
		k8sClient      client.Client
		codebaseBranch *codebaseApi.CodebaseBranch
		codebaseName   string
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
		want    func(t *testing.T, k8sClient client.Client)
	}{
		{
			name: "successfully add labels",
			args: args{
				k8sClient: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-branch",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				}).Build(),
				codebaseBranch: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-branch",
						Namespace:       "default",
						ResourceVersion: "999",
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				},
				codebaseName: "test-codebase",
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				cb := &codebaseApi.CodebaseBranch{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-branch", Namespace: "default"}, cb)
				assert.NoError(t, err)
				assert.Equal(t, "test-codebase", cb.Labels[LabelCodebaseName])
				assert.Equal(t, "test-codebase", cb.Labels[codebaseApi.CodebaseLabel])
			},
		},
		{
			name: "some labels already exist",
			args: args{
				k8sClient: fake.NewClientBuilder().WithObjects(&codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-branch2",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				}).WithScheme(scheme).Build(),
				codebaseBranch: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-branch2",
						Namespace: "default",
						Labels: map[string]string{
							LabelCodebaseName: "test-codebase",
						},
						ResourceVersion: "999",
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				},
				codebaseName: "test-codebase",
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				cb := &codebaseApi.CodebaseBranch{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-branch2", Namespace: "default"}, cb)
				assert.NoError(t, err)
				assert.Equal(t, "test-codebase", cb.Labels[LabelCodebaseName])
				assert.Equal(t, "test-codebase", cb.Labels[codebaseApi.CodebaseLabel])
			},
		},
		{
			name: "no update needed",
			args: args{
				k8sClient: fake.NewClientBuilder().WithObjects(&codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-branch3",
						Namespace: "default",
						Labels: map[string]string{
							"app.edp.epam.com/codebaseName": "test-codebase",
							"app.edp.epam.com/codebase":     "test-codebase",
							"app.edp.epam.com/git-branch":   "main",
						},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				}).WithScheme(scheme).Build(),
				codebaseBranch: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-branch3",
						Namespace: "default",
						Labels: map[string]string{
							"app.edp.epam.com/codebaseName": "test-codebase",
							"app.edp.epam.com/codebase":     "test-codebase",
							"app.edp.epam.com/git-branch":   "main",
						},
						ResourceVersion: "999",
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				},
				codebaseName: "test-codebase",
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				cb := &codebaseApi.CodebaseBranch{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-branch3", Namespace: "default"}, cb)
				assert.NoError(t, err)
				assert.Equal(t, "test-codebase", cb.Labels[LabelCodebaseName])
				assert.Equal(t, "test-codebase", cb.Labels[codebaseApi.CodebaseLabel])
			},
		},
		{
			name: "error updating resource",
			args: args{
				k8sClient: fake.NewClientBuilder().WithObjects().WithScheme(scheme).Build(),
				codebaseBranch: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-branch",
						Namespace:       "default",
						ResourceVersion: "999",
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "main",
					},
				},
				codebaseName: "test-codebase",
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddCodebaseLabel(
				context.Background(),
				tt.args.k8sClient,
				tt.args.codebaseBranch,
				tt.args.codebaseName,
			)
			tt.wantErr(t, err)

			if tt.want != nil {
				tt.want(t, tt.args.k8sClient)
			}
		})
	}
}
