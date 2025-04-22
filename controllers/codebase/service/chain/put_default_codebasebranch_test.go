package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutDefaultCodeBaseBranch_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	err := codebaseApi.AddToScheme(scheme)
	require.NoError(t, err)

	type fields struct {
		client client.Client
	}

	type args struct {
		codebase *codebaseApi.Codebase
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should put default codebase branch",
			fields: fields{
				client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			},
			args: args{
				codebase: &codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "test-ns",
						Name:      "test",
					},
					Spec: codebaseApi.CodebaseSpec{
						DefaultBranch: "master",
						Versioning: codebaseApi.Versioning{
							Type: codebaseApi.VersioningTypDefault,
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "should put default codebase branch with version",
			fields: fields{
				client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			},
			args: args{
				codebase: &codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "test-ns",
						Name:      "test",
					},
					Spec: codebaseApi.CodebaseSpec{
						DefaultBranch: "master",
						Versioning: codebaseApi.Versioning{
							Type: "semver",
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "codebase branch already exists",
			fields: fields{
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects([]client.Object{
						&codebaseApi.CodebaseBranch{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-master",
								Namespace: "test-ns",
							},
						},
					}...).
					Build(),
			},
			args: args{
				codebase: &codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "test-ns",
						Name:      "test",
					},
					Spec: codebaseApi.CodebaseSpec{
						DefaultBranch: "master",
						Versioning: codebaseApi.Versioning{
							Type: "semver",
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &PutDefaultCodeBaseBranch{
				client: tt.fields.client,
			}

			ctx := context.Background()

			err := s.ServeRequest(ctx, tt.args.codebase)

			tt.wantErr(t, err)

			codebaseBranch := &codebaseApi.CodebaseBranch{}

			err = tt.fields.client.Get(
				ctx,
				client.ObjectKey{
					Namespace: tt.args.codebase.Namespace,
					Name:      fmt.Sprintf("%s-%s", tt.args.codebase.Name, tt.args.codebase.Spec.DefaultBranch),
				},
				codebaseBranch,
			)

			assert.NoError(t, err)
		})
	}
}
