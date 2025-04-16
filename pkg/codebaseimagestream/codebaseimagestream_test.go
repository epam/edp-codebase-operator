package codebaseimagestream

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestGetLastTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tags    []codebaseApi.Tag
		want    codebaseApi.Tag
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return latest tag",
			tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-4",
					Created: "2022-04-11T12:00:00Z",
				},
				{
					Name:    "master-0.0.1-6",
					Created: "2022-04-12T12:54:04Z",
				},
			},
			want: codebaseApi.Tag{
				Name:    "master-0.0.1-6",
				Created: "2022-04-12T12:54:04Z",
			},
			wantErr: assert.NoError,
		},
		{
			name: "should skip tag with invalid created time",
			tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-6",
					Created: "2022-04-12T12:54:04Z",
				},
				{
					Name:    "master-0.0.1-7",
					Created: "2022-04-12",
				},
			},
			want: codebaseApi.Tag{
				Name:    "master-0.0.1-6",
				Created: "2022-04-12T12:54:04Z",
			},
			wantErr: assert.NoError,
		},
		{
			name:    "should return error if latest tag is not found",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetLastTag(tt.tags, logr.Discard())
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetCodebaseImageStreamByCodebaseBaseBranchName(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	type args struct {
		k8sCl              func(t *testing.T) client.Client
		codebaseBranchName string
	}

	tests := []struct {
		name    string
		args    args
		want    require.ValueAssertionFunc
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				k8sCl: func(t *testing.T) client.Client {
					obj := &codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-branch",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "test-branch",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build()
				},
				codebaseBranchName: "test-branch",
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
		{
			name: "not found",
			args: args{
				k8sCl: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
				codebaseBranchName: "non-existent-branch",
			},
			want: require.Nil,
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), "CodebaseImageStream not found")
			},
		},
		{
			name: "multipleFound",
			args: args{
				k8sCl: func(t *testing.T) client.Client {
					obj1 := &codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-branch-1",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "test-branch",
							},
						},
					}
					obj2 := &codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-branch-2",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "test-branch",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj1, obj2).Build()
				},
				codebaseBranchName: "test-branch",
			},
			want: require.Nil,
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), "multiple CodebaseImageStream found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetCodebaseImageStreamByCodebaseBaseBranchName(
				context.Background(),
				tt.args.k8sCl(t),
				tt.args.codebaseBranchName,
				"default",
			)
			tt.wantErr(t, err)
		})
	}
}
