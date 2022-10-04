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

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

func TestPutDefaultCodeBaseBranch_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	err := codebaseApi.AddToScheme(scheme)
	require.NoError(t, err)

	tests := []struct {
		name                     string
		objects                  []client.Object
		codebase                 *codebaseApi.Codebase
		wantErr                  assert.ErrorAssertionFunc
		wantGetCodeBaseBranchErr assert.ErrorAssertionFunc
	}{
		{
			name: "should put default codebase branch",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test",
				},
				Spec: codebaseApi.CodebaseSpec{
					DefaultBranch: "master",
					Versioning: codebaseApi.Versioning{
						Type: codebaseApi.Default,
					},
				},
			},
			wantErr:                  assert.NoError,
			wantGetCodeBaseBranchErr: assert.NoError,
		},
		{
			name: "should put default codebase branch with version",
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
			wantErr:                  assert.NoError,
			wantGetCodeBaseBranchErr: assert.NoError,
		},
		{
			name:    "codebase branch already exists",
			objects: []client.Object{&codebaseApi.CodebaseBranch{ObjectMeta: metaV1.ObjectMeta{Name: "test-master", Namespace: "test-ns"}}},
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
			wantErr:                  assert.NoError,
			wantGetCodeBaseBranchErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			s := NewPutDefaultCodeBaseBranch(k8sClient)
			ctx := context.Background()

			tt.wantErr(t, s.ServeRequest(ctx, tt.codebase))

			codebaseBranch := &codebaseApi.CodebaseBranch{}
			err = k8sClient.Get(
				ctx,
				client.ObjectKey{
					Namespace: tt.codebase.Namespace,
					Name:      fmt.Sprintf("%s-%s", tt.codebase.Name, tt.codebase.Spec.DefaultBranch),
				},
				codebaseBranch,
			)

			tt.wantGetCodeBaseBranchErr(t, err)
		})
	}
}
