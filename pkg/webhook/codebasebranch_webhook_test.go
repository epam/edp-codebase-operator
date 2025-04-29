package webhook

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
)

func TestCodebaseBranchValidationWebhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	type fields struct {
		client func(t *testing.T) client.Client
	}

	type args struct {
		obj runtime.Object
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should return nil when the object is not a CodebaseBranch",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
			},
			args: args{
				obj: &codebaseApi.Codebase{},
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to list CodebaseBranch CRs",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().Build()
				},
			},
			args: args{
				obj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "test-codebase-test-branch2",
						Labels: map[string]string{
							codebaseApi.CodebaseLabel:   "test-codebase",
							codebaseApi.BranchHashLabel: codebasebranch.MakeGitBranchHash("test-branch"),
						},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						CodebaseName: "test-codebase",
						BranchName:   "test-branch",
					},
				},
			},
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "failed to list CodebaseBranch CRs")
			},
		},
		{
			name: "should return error when CodebaseBranch with the same codebase name and branch name exists",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
						&codebaseApi.CodebaseBranch{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-namespace",
								Name:      "test-codebase-test-branch",
								Labels: map[string]string{
									codebaseApi.CodebaseLabel:   "test-codebase",
									codebaseApi.BranchHashLabel: codebasebranch.MakeGitBranchHash("test-branch"),
								},
							},
							Spec: codebaseApi.CodebaseBranchSpec{
								CodebaseName: "test-codebase",
								BranchName:   "test-branch",
							},
						},
					).Build()
				},
			},
			args: args{
				obj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "test-codebase-test-branch2",
						Labels: map[string]string{
							codebaseApi.CodebaseLabel:   "test-codebase",
							codebaseApi.BranchHashLabel: codebasebranch.MakeGitBranchHash("test-branch"),
						},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						CodebaseName: "test-codebase",
						BranchName:   "test-branch",
					},
				},
			},
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "CodebaseBranch CR with the same codebase name and branch name already exists")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewCodebaseBranchValidationWebhook(tt.fields.client(t), logr.Discard())
			gotWarnings, err := r.ValidateCreate(t.Context(), tt.args.obj)

			assert.Nil(t, gotWarnings)
			tt.wantErr(t, err)
		})
	}
}

func TestCodebaseBranchValidationWebhook_ValidateUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	type fields struct {
		client func(t *testing.T) client.Client
	}

	type args struct {
		oldObj runtime.Object
		newObj runtime.Object
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should return error when resource protection check fails on update",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
			},
			args: args{
				oldObj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Labels: map[string]string{
							protectedLabel: updateOperation,
						},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						CodebaseName: "test-codebase",
						Release:      false,
					},
				},
				newObj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Labels: map[string]string{
							protectedLabel: updateOperation,
						},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						CodebaseName: "test-codebase",
						Release:      true,
					},
				},
			},
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "esource contains label that protects it from modification")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewCodebaseBranchValidationWebhook(tt.fields.client(t), logr.Discard())
			gotWarnings, err := r.ValidateUpdate(t.Context(), tt.args.oldObj, tt.args.newObj)

			assert.Nil(t, gotWarnings)
			tt.wantErr(t, err)
		})
	}
}

func TestCodebaseBranchValidationWebhook_ValidateDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	type fields struct {
		client func(t *testing.T) client.Client
	}

	type args struct {
		obj runtime.Object
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should return error when resource protection check fails on delete",
			fields: fields{
				client: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
			},
			args: args{
				obj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Labels: map[string]string{
							protectedLabel: deleteOperation,
						},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						CodebaseName: "test-codebase",
						BranchName:   "test-branch",
					},
				},
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewCodebaseBranchValidationWebhook(tt.fields.client(t), logr.Discard())
			gotWarnings, err := r.ValidateDelete(t.Context(), tt.args.obj)

			assert.Nil(t, gotWarnings)
			tt.wantErr(t, err)
		})
	}
}
