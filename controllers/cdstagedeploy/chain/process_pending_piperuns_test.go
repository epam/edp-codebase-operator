package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		{
			name: "pipelineRun completed, CDStageDeploy should be completed",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-dev-deploy",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Stage:    "dev",
					Pipeline: "pipeline",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusRunning,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				run := &tektonpipelineApi.PipelineRun{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "test-pipeline-run",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdStageDeployLabel: "pipeline-dev-deploy",
							codebaseApi.CdPipelineLabel:    "pipeline",
							codebaseApi.CdStageLabel:       "pipeline-dev",
						},
					},
				}
				run.Status.MarkSucceeded("done", "done")

				return fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(run).WithObjects(run).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, stageDeploy *codebaseApi.CDStageDeploy) {
				require.Equal(t, codebaseApi.CDStageDeployStatusCompleted, stageDeploy.Status.Status)
			},
		},
		{
			name: "should start pending pipelineRun",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-dev-deploy",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Stage:    "dev",
					Pipeline: "pipeline",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusInQueue,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				run := &tektonpipelineApi.PipelineRun{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "test-pipeline-run",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdStageDeployLabel: "pipeline-dev-deploy",
							codebaseApi.CdPipelineLabel:    "pipeline",
							codebaseApi.CdStageLabel:       "pipeline-dev",
						},
					},
					Spec: tektonpipelineApi.PipelineRunSpec{
						Status: tektonpipelineApi.PipelineRunSpecStatusPending,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(run).WithObjects(run).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, stageDeploy *codebaseApi.CDStageDeploy) {
				require.Equal(t, codebaseApi.CDStageDeployStatusRunning, stageDeploy.Status.Status)
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
