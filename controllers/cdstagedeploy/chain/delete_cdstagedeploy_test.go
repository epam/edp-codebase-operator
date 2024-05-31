package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestDeleteCDStageDeploy_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stageDeploy *codebaseApi.CDStageDeploy
		objects     []client.Object
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name: "should delete CDStageDeploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			objects: []client.Object{
				&codebaseApi.CDStageDeploy{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Status: codebaseApi.CDStageDeployStatus{
						Status: codebaseApi.CDStageDeployStatusCompleted,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "should ignore error if CDStageDeploy doesn't exist",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusCompleted,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "should ignore if CDStageDeploy has not been completed yet",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &DeleteCDStageDeploy{
				client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
			}

			tt.wantErr(t, h.ServeRequest(context.Background(), tt.stageDeploy))
		})
	}
}
