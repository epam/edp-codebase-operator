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

func TestResolveStatus_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, tektonpipelineApi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stageDeploy *codebaseApi.CDStageDeploy
		k8sClient   func(t *testing.T) client.Client
		wantErr     require.ErrorAssertionFunc
		wantStatus  string
	}{
		{
			name: "should skip failed CDStageDeploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusFailed,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusFailed,
		},
		{
			name: "should skip pending CDStageDeploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusPending,
		},
		{
			name: "have running pipeline runs",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusRunning,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				piperun := &tektonpipelineApi.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							"app.edp.epam.com/cdpipeline": "app1",
							"app.edp.epam.com/cdstage":    "app1-dev",
						},
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(piperun).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusRunning,
		},
		{
			name: "all pipeline runs completed",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusRunning,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusCompleted,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewResolveStatus(tt.k8sClient(t))

			tt.wantErr(t, r.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stageDeploy))
			require.Equal(t, tt.wantStatus, tt.stageDeploy.Status.Status)
		})
	}
}
