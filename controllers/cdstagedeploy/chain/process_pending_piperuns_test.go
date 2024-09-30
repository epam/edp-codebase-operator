package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestProcessPendingPipeRuns_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, tektonpipelineApi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stageDeploy *codebaseApi.CDStageDeploy
		k8sClient   func(t *testing.T) client.Client
		wantErr     require.ErrorAssertionFunc
		want        func(t *testing.T, stageDeploy *codebaseApi.CDStageDeploy)
	}{
		{
			name: "skip processing pending PipelineRuns for CDStageDeploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusCompleted,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, stageDeploy *codebaseApi.CDStageDeploy) {
				require.Equal(t, codebaseApi.CDStageDeployStatusCompleted, stageDeploy.Status.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewProcessPendingPipeRuns(tt.k8sClient(t))

			tt.wantErr(t, r.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stageDeploy))

			if tt.want != nil {
				tt.want(t, tt.stageDeploy)
			}
		})
	}
}
