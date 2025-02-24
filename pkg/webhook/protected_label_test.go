package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestProtectedLabelValidationWebhook_ValidateUpdate(t *testing.T) {
	type args struct {
		oldObj runtime.Object
		newObj runtime.Object
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should fail if resource contains label that protects it from modification",
			args: args{
				oldObj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{protectedLabel: updateOperation},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "test-branch",
					},
				},
				newObj: &codebaseApi.CodebaseBranch{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{protectedLabel: updateOperation},
					},
					Spec: codebaseApi.CodebaseBranchSpec{
						BranchName: "test-branch2",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "resource contains label that protects it from modification")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &ProtectedLabelValidationWebhook{}
			_, err := pr.ValidateUpdate(context.Background(), tt.args.oldObj, tt.args.newObj)
			tt.wantErr(t, err)
		})
	}
}

func TestProtectedLabelValidationWebhook_ValidateDelete(t *testing.T) {
	type args struct {
		obj runtime.Object
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should fail if resource contains label that protects it from modification",
			args: args{
				obj: &codebaseApi.GitServer{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{protectedLabel: deleteOperation},
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost: "test-git-host",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "resource contains label that protects it from deletion")
			},
		},
		{
			name: "should pass if resource doesn't contain label that protects it from modification",
			args: args{
				obj: &codebaseApi.GitServer{
					Spec: codebaseApi.GitServerSpec{
						GitHost: "test-git-host",
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &ProtectedLabelValidationWebhook{}
			_, err := pr.ValidateDelete(context.Background(), tt.args.obj)

			tt.wantErr(t, err)
		})
	}
}
