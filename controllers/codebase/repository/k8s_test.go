package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestNewK8SCodebaseRepository(t *testing.T) {
	t.Parallel()

	type args struct {
		c  client.Client
		cr *codebaseApi.Codebase
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should succeed",
			args: args{
				c: fake.NewClientBuilder().Build(),
				cr: &codebaseApi.Codebase{
					TypeMeta: metaV1.TypeMeta{
						Kind:       "fake-kind",
						APIVersion: "v.test",
					},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec:       codebaseApi.CodebaseSpec{},
					Status:     codebaseApi.CodebaseStatus{},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := &K8SCodebaseRepository{
				client: tt.args.c,
				cr:     tt.args.cr,
			}

			got := NewK8SCodebaseRepository(tt.args.c, tt.args.cr)

			assert.Equal(t, want, got)
		})
	}
}

func TestK8SCodebaseRepository_SelectProjectStatusValue(t *testing.T) {
	t.Parallel()

	type fields struct {
		client client.Client
		cr     *codebaseApi.Codebase
	}

	type args struct {
		in1 string
		in2 string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should succeed",
			fields: fields{
				client: nil,
				cr: &codebaseApi.Codebase{
					TypeMeta:   metaV1.TypeMeta{},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec:       codebaseApi.CodebaseSpec{},
					Status: codebaseApi.CodebaseStatus{
						Git: "git status",
					},
				},
			},
			args:    args{},
			want:    "git status",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &K8SCodebaseRepository{
				client: tt.fields.client,
				cr:     tt.fields.cr,
			}

			ctx := context.Background()

			got, err := r.SelectProjectStatusValue(ctx, tt.args.in1, tt.args.in2)

			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestK8SCodebaseRepository_UpdateProjectStatusValue(t *testing.T) {
	t.Parallel()

	type fields struct {
		client client.Client
		cr     *codebaseApi.Codebase
	}

	type args struct {
		gitStatus string
		in2       string
		in3       string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should succeed",
			fields: fields{
				client: fake.NewClientBuilder().Build(),
				cr: &codebaseApi.Codebase{
					TypeMeta: metaV1.TypeMeta{
						Kind:       "test-kind",
						APIVersion: "v.test",
					},
					ObjectMeta: metaV1.ObjectMeta{
						Name: "fake-name",
					},
					Spec: codebaseApi.CodebaseSpec{},
					Status: codebaseApi.CodebaseStatus{
						Git: "git status",
					},
				},
			},
			args: args{
				gitStatus: "new git status",
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &K8SCodebaseRepository{
				client: tt.fields.client,
				cr:     tt.fields.cr,
			}

			ctx := context.Background()

			if r.client != nil {
				err := codebaseApi.AddToScheme(r.client.Scheme())
				assert.NoError(t, err)

				if r.cr != nil {
					err = r.client.Create(ctx, tt.fields.cr)
					assert.NoError(t, err)
				}
			}

			err := r.UpdateProjectStatusValue(ctx, tt.args.gitStatus, tt.args.in2, tt.args.in3)

			tt.wantErr(t, err)
		})
	}
}
