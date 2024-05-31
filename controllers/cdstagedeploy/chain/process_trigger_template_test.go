package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tektonTriggersApi "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

var (
	pipelineRunTemplate = []byte(`{
	  "apiVersion": "tekton.dev/v1",
	  "kind": "PipelineRun",
	  "metadata": {
		"generateName": "deploy-$(tt.params.CDPIPELINE)-$(tt.params.CDSTAGE)-",
		"labels": {
		  "app.edp.epam.com/cdpipeline": "$(tt.params.CDPIPELINE)",
		  "app.edp.epam.com/cdstage": "$(tt.params.CDSTAGE)",
		  "app.edp.epam.com/pipelinetype": "deploy"
		}
	  },
	  "spec": {
		"params": [
		  {
			"name": "APPLICATIONS_PAYLOAD",
			"value": "$(tt.params.APPLICATIONS_PAYLOAD)"
		  },
		  {
			"name": "CDSTAGE",
			"value": "$(tt.params.CDSTAGE)"
		  },
		  {
			"name": "CDPIPELINE",
			"value": "$(tt.params.CDPIPELINE)"
		  },
          {
			"name": "KUBECONFIG_SECRET_NAME",
			"value": "$(tt.params.KUBECONFIG_SECRET_NAME)"
          }
		],
		"pipelineRef": {
		  "name": "cd-stage-deploy"
		},
		"taskRunTemplate": {
		  "serviceAccountName": "tekton"
		},
		"timeouts": {
		  "pipeline": "1h00m0s"
		}
	  }
	}`)
)

func TestProcessTriggerTemplate_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineAPi.AddToScheme(scheme))
	require.NoError(t, tektonTriggersApi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stageDeploy *codebaseApi.CDStageDeploy
		k8sClient   func(t *testing.T) client.Client
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "should create trigger template resources",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev",
							},
							Spec: pipelineAPi.CDPipelineSpec{
								Name:               "dev",
								Applications:       []string{"app1", "app2"},
								InputDockerStreams: []string{"app1-main", "app2-main"},
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "app1-main",
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								Tags: []codebaseApi.Tag{
									{
										Name:    "main-0.0.1",
										Created: "2024-03-04T11:36:26Z",
									},
									{
										Name:    "main-0.0.2",
										Created: "2024-03-05T11:00:00Z",
									},
								},
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "app2-main",
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								Tags: []codebaseApi.Tag{
									{
										Name:    "main-0.0.3",
										Created: "2024-03-04T11:36:26Z",
									},
								},
							},
						},
						&pipelineAPi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev-qa",
							},
							Spec: pipelineAPi.StageSpec{
								Name:            "qa",
								TriggerTemplate: "auto-deploy",
								ClusterName:     pipelineAPi.InCluster,
							},
						},
						&tektonTriggersApi.TriggerTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "auto-deploy",
							},
							Spec: tektonTriggersApi.TriggerTemplateSpec{
								ResourceTemplates: []tektonTriggersApi.TriggerResourceTemplate{
									{
										RawExtension: runtime.RawExtension{
											Raw: pipelineRunTemplate,
										},
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "trigger template doesn't contain resources, skip processing",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev",
							},
							Spec: pipelineAPi.CDPipelineSpec{
								Name:               "dev",
								Applications:       []string{"app1"},
								InputDockerStreams: []string{"app1-main"},
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "app1-main",
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								Tags: []codebaseApi.Tag{
									{
										Name:    "main-0.0.1",
										Created: "2024-03-04T11:36:26Z",
									},
								},
							},
						},
						&pipelineAPi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev-qa",
							},
							Spec: pipelineAPi.StageSpec{
								Name:            "qa",
								TriggerTemplate: "auto-deploy",
							},
						},
						&tektonTriggersApi.TriggerTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "auto-deploy",
							},
							Spec: tektonTriggersApi.TriggerTemplateSpec{},
						},
					).
					Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "trigger template not found",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev",
							},
							Spec: pipelineAPi.CDPipelineSpec{
								Name:               "dev",
								Applications:       []string{"app1"},
								InputDockerStreams: []string{"app1-main"},
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "app1-main",
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								Tags: []codebaseApi.Tag{
									{
										Name:    "main-0.0.1",
										Created: "2024-03-04T11:36:26Z",
									},
								},
							},
						},
						&pipelineAPi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev-qa",
							},
							Spec: pipelineAPi.StageSpec{
								Name:            "qa",
								TriggerTemplate: "auto-deploy",
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get TriggerTemplate")
			},
		},
		{
			name: "stage not found",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev",
							},
							Spec: pipelineAPi.CDPipelineSpec{
								Name:               "dev",
								Applications:       []string{"app1"},
								InputDockerStreams: []string{"app1-main"},
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "app1-main",
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								Tags: []codebaseApi.Tag{
									{
										Name:    "main-0.0.1",
										Created: "2024-03-04T11:36:26Z",
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get Stage")
			},
		},
		{
			name: "codebase image stream doesn't contain tags, skip processing",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev",
							},
							Spec: pipelineAPi.CDPipelineSpec{
								Name:               "dev",
								Applications:       []string{"app1"},
								InputDockerStreams: []string{"app1-main"},
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "app1-main",
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{},
						},
					).
					Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "codebase image stream not found",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "default",
								Name:      "dev",
							},
							Spec: pipelineAPi.CDPipelineSpec{
								Name:               "dev",
								Applications:       []string{"app1"},
								InputDockerStreams: []string{"app1-main"},
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get app1-main CodebaseImageStream")
			},
		},
		{
			name: "pipeline not found",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusPending,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get CDPipeline")
			},
		},
		{
			name: "skip processing trigger template for auto-deploy if status is completed",
			stageDeploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "dev-qa",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "dev",
					Stage:    "qa",
				},
				Status: codebaseApi.CDStageDeployStatus{
					Status: codebaseApi.CDStageDeployStatusCompleted,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewProcessTriggerTemplate(tt.k8sClient(t))
			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stageDeploy)

			tt.wantErr(t, err)
		})
	}
}
