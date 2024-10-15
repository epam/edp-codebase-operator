package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/autodeploy"
	autodeploymocks "github.com/epam/edp-codebase-operator/v2/pkg/autodeploy/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/tektoncd"
	tektoncdmocks "github.com/epam/edp-codebase-operator/v2/pkg/tektoncd/mocks"
)

func TestProcessTriggerTemplate_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	type fields struct {
		k8sClient                 func(t *testing.T) client.Client
		triggerTemplateManager    func(t *testing.T) tektoncd.TriggerTemplateManager
		autoDeployStrategyManager func(t *testing.T) autodeploy.Manager
	}

	tests := []struct {
		name        string
		stageDeploy *codebaseApi.CDStageDeploy
		fields      fields
		wantErr     require.ErrorAssertionFunc
		want        func(t *testing.T, d *codebaseApi.CDStageDeploy)
	}{
		{
			name: "should process TriggerTemplate for auto-deploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return([]byte("raw resource"), nil)
					m.On(
						"CreatePipelineRun",
						mock.Anything,
						"default",
						"test",
						[]byte("raw resource"),
						[]byte("{app1: 1.0}"),
						[]byte("dev"),
						[]byte("pipe1"),
						[]byte("cluster-secret"),
					).Return(nil)

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					m := autodeploymocks.NewMockManager(t)

					m.On("GetAppPayloadForAllLatestStrategy", mock.Anything, mock.Anything).
						Return(json.RawMessage("{app1: 1.0}"), nil)

					return m
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, d *codebaseApi.CDStageDeploy) {
				assert.Equal(t, codebaseApi.CDStageDeployStatusRunning, d.Status.Status)
			},
		},
		{
			name: "should process TriggerTemplate for Auto-stable auto-deploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline:    "pipe1",
					Stage:       "dev",
					TriggerType: pipelineApi.TriggerTypeAutoStable,
					Tag: codebaseApi.CodebaseTag{
						Codebase: "app1",
						Tag:      "1.0",
					},
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return([]byte("raw resource"), nil)
					m.On(
						"CreatePipelineRun",
						mock.Anything,
						"default",
						"test",
						[]byte("raw resource"),
						[]byte("{app1: 1.0}"),
						[]byte("dev"),
						[]byte("pipe1"),
						[]byte("cluster-secret"),
					).Return(nil)

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					m := autodeploymocks.NewMockManager(t)

					m.On(
						"GetAppPayloadForCurrentWithStableStrategy",
						mock.Anything,
						codebaseApi.CodebaseTag{
							Codebase: "app1",
							Tag:      "1.0",
						},
						mock.Anything,
						mock.Anything,
					).
						Return(json.RawMessage("{app1: 1.0}"), nil)

					return m
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, d *codebaseApi.CDStageDeploy) {
				assert.Equal(t, codebaseApi.CDStageDeployStatusRunning, d.Status.Status)
			},
		},
		{
			name: "failed to create PipelineRun",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return([]byte("raw resource"), nil)
					m.On(
						"CreatePipelineRun",
						mock.Anything,
						"default",
						"test",
						[]byte("raw resource"),
						[]byte("{app1: 1.0}"),
						[]byte("dev"),
						[]byte("pipe1"),
						[]byte("cluster-secret"),
					).Return(errors.New("failed to create PipelineRun"))

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					m := autodeploymocks.NewMockManager(t)

					m.On("GetAppPayloadForAllLatestStrategy", mock.Anything, mock.Anything).
						Return(json.RawMessage("{app1: 1.0}"), nil)

					return m
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create PipelineRun")
			},
		},
		{
			name: "failed to get app payload for all latest strategy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return([]byte("raw resource"), nil)

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					m := autodeploymocks.NewMockManager(t)

					m.On("GetAppPayloadForAllLatestStrategy", mock.Anything, mock.Anything).
						Return(nil, errors.New("failed to get app payload"))

					return m
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get app payload")
			},
		},
		{
			name: "last tag not found",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return([]byte("raw resource"), nil)

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					m := autodeploymocks.NewMockManager(t)

					m.On("GetAppPayloadForAllLatestStrategy", mock.Anything, mock.Anything).
						Return(nil, fmt.Errorf("failed to get app payload: %w", autodeploy.ErrLasTagNotFound))

					return m
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, d *codebaseApi.CDStageDeploy) {
				assert.Equal(t, codebaseApi.CDStageDeployStatusCompleted, d.Status.Status)
			},
		},
		{
			name: "failed to get raw resource from trigger template",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return(nil, errors.New("failed to get raw resource"))

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					return autodeploymocks.NewMockManager(t)
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get raw resource")
			},
		},
		{
			name: "no resource templates found in the trigger template",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
							&pipelineApi.Stage{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1-dev",
									Namespace: "default",
								},
								Spec: pipelineApi.StageSpec{
									TriggerTemplate: "trigger1",
									Name:            "dev",
									ClusterName:     "cluster-secret",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					m := tektoncdmocks.NewMockTriggerTemplateManager(t)

					m.On("GetRawResourceFromTriggerTemplate", mock.Anything, "trigger1", "default").
						Return(nil, fmt.Errorf("failed to get TriggerTemplate: %w", tektoncd.ErrEmptyTriggerTemplateResources))

					return m
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					return autodeploymocks.NewMockManager(t)
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, d *codebaseApi.CDStageDeploy) {
				assert.Equal(t, codebaseApi.CDStageDeployStatusCompleted, d.Status.Status)
			},
		},
		{
			name: "failed to get Stage",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&pipelineApi.CDPipeline{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "pipe1",
									Namespace: "default",
								},
								Spec: pipelineApi.CDPipelineSpec{
									Name: "pipe1",
								},
							},
						).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					return tektoncdmocks.NewMockTriggerTemplateManager(t)
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					return autodeploymocks.NewMockManager(t)
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get Stage")
			},
		},
		{
			name: "failed to get CDPipeline",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "pipe1",
					Stage:    "dev",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					return tektoncdmocks.NewMockTriggerTemplateManager(t)
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					return autodeploymocks.NewMockManager(t)
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get CDPipeline")
			},
		},
		{
			name: "skip processing TriggerTemplate for auto-deploy",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusInQueue,
				},
			},
			fields: fields{
				k8sClient: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						Build()
				},
				triggerTemplateManager: func(t *testing.T) tektoncd.TriggerTemplateManager {
					return tektoncdmocks.NewMockTriggerTemplateManager(t)
				},
				autoDeployStrategyManager: func(t *testing.T) autodeploy.Manager {
					return autodeploymocks.NewMockManager(t)
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, d *codebaseApi.CDStageDeploy) {
				assert.Equal(t, codebaseApi.CDStageDeployStatusInQueue, d.Status.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewProcessTriggerTemplate(
				tt.fields.k8sClient(t),
				tt.fields.triggerTemplateManager(t),
				tt.fields.autoDeployStrategyManager(t),
			)

			tt.wantErr(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stageDeploy))
			if tt.want != nil {
				tt.want(t, tt.stageDeploy)
			}
		})
	}
}
