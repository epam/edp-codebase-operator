package chain

import (
	"context"
	"testing"
	"time"

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
			name: "failed CDStageDeploy should be pending to retry",
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
			name: "pending CDStageDeploy should be pending if no pipeline runs",
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
			name: "pending CDStageDeploy should be in queue if not all pipeline runs completed",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				piperun := &tektonpipelineApi.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(piperun).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusInQueue,
		},
		{
			name: "running CDStageDeploy should be running if pipeline run is running",
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
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(piperun).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusRunning,
		},
		{
			name: "running CDStageDeploy should be completed after all pipeline runs completed",
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
		{
			name: "queued CDStageDeploy should be pending after all pipeline runs completed",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusInQueue,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				d := &codebaseApi.CDStageDeploy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
					},
					Spec: codebaseApi.CDStageDeploySpec{
						Pipeline: "app1",
						Stage:    "dev",
					},
					Status: codebaseApi.CDStageDeployStatus{
						Status: codebaseApi.CDStageDeployStatusInQueue,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(d).WithStatusSubresource(d).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusPending,
		},
		{
			name: "queued CDStageDeploy should be queued if it is not first in queue",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusInQueue,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				d1 := &codebaseApi.CDStageDeploy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
						CreationTimestamp: metav1.NewTime(time.Now()),
					},
					Spec: codebaseApi.CDStageDeploySpec{
						Pipeline: "app1",
						Stage:    "dev",
					},
					Status: codebaseApi.CDStageDeployStatus{
						Status: codebaseApi.CDStageDeployStatusInQueue,
					},
				}

				d2 := &codebaseApi.CDStageDeploy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
						CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
					},
					Spec: codebaseApi.CDStageDeploySpec{
						Pipeline: "app1",
						Stage:    "dev",
					},
					Status: codebaseApi.CDStageDeployStatus{
						Status: codebaseApi.CDStageDeployStatusInQueue,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(d1, d2).WithStatusSubresource(d1, d2).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusInQueue,
		},
		{
			name: "queued CDStageDeploy should be queued if not all pipeline runs completed",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusInQueue,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				piperun := &tektonpipelineApi.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(piperun).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusInQueue,
		},
		{
			name: "queued CDStageDeploy should be queued if another CDStageDeploy is running",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusInQueue,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				stageDeploy := &codebaseApi.CDStageDeploy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "running",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CdPipelineLabel: "app1",
							codebaseApi.CdStageLabel:    "app1-dev",
						},
					},
					Spec: codebaseApi.CDStageDeploySpec{
						Pipeline: "app1",
						Stage:    "dev",
					},
					Status: codebaseApi.CDStageDeployStatus{
						Status: codebaseApi.CDStageDeployStatusRunning,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(stageDeploy).Build()
			},
			wantErr:    require.NoError,
			wantStatus: codebaseApi.CDStageDeployStatusInQueue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewResolveStatus(tt.k8sClient(t))

			tt.wantErr(t, r.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stageDeploy))
			require.Equal(t, tt.wantStatus, tt.stageDeploy.Status.Status)
		})
	}
}
