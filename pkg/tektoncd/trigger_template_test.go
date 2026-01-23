package tektoncd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonTriggersApi "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

func TestTektonTriggerTemplateManager_GetRawResourceFromTriggerTemplate(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, tektonTriggersApi.AddToScheme(scheme))

	tests := []struct {
		name      string
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "get raw resource successfully",
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&tektonTriggersApi.TriggerTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app-template",
							Namespace: "default",
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
					}).
					Build()
			},
			want:    string(pipelineRunTemplate),
			wantErr: require.NoError,
		},
		{
			name: "raw resource is empty",
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&tektonTriggersApi.TriggerTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app-template",
							Namespace: "default",
						},
						Spec: tektonTriggersApi.TriggerTemplateSpec{},
					}).
					Build()
			},
			want: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrEmptyTriggerTemplateResources)
			},
		},
		{
			name: "failed to get trigger template",
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			},
			want: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get TriggerTemplate")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewTektonTriggerTemplateManager(tt.k8sClient(t))
			got, err := h.GetRawResourceFromTriggerTemplate(context.Background(), "app-template", "default")

			tt.wantErr(t, err)

			if tt.want != "" {
				require.JSONEq(t, tt.want, string(got))
			}
		})
	}
}

func TestTektonTriggerTemplateManager_CreatePipelineRun(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, tektonpipelineApi.AddToScheme(scheme))
	require.NoError(t, tektonTriggersApi.AddToScheme(scheme))

	type args struct {
		ns                string
		cdStageDeployName string
		rawPipeRun        []byte
		appPayload        []byte
		stage             []byte
		pipeline          []byte
		clusterSecret     []byte
	}

	tests := []struct {
		name      string
		args      args
		k8sClient func(t *testing.T) client.Client
		wantErr   require.ErrorAssertionFunc
		want      func(t *testing.T, k8sCl client.Client)
	}{
		{
			name: "create pipeline run successfully",
			args: args{
				ns:                "default",
				cdStageDeployName: "deploy-app",
				rawPipeRun:        pipelineRunTemplate,
				appPayload:        []byte(`{"app1":"1.1", "app2":"2.0"}`),
				stage:             []byte("dev"),
				pipeline:          []byte("pipeline-1"),
				clusterSecret:     []byte("cl-secret"),
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				l := &tektonpipelineApi.PipelineRunList{}
				require.NoError(t, k8sCl.List(context.Background(), l, client.InNamespace("default")))
				require.Len(t, l.Items, 1)

				run := l.Items[0]

				require.Len(t, run.Spec.Params, 4)

				for _, p := range run.Spec.Params {
					switch p.Name {
					case "APPLICATIONS_PAYLOAD":
						require.Equal(t, `{"app1":"1.1", "app2":"2.0"}`, p.Value.StringVal)
					case "CDSTAGE":
						require.Equal(t, "dev", p.Value.StringVal)
					case "CDPIPELINE":
						require.Equal(t, "pipeline-1", p.Value.StringVal)
					case "KUBECONFIG_SECRET_NAME":
						require.Equal(t, "cl-secret", p.Value.StringVal)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sCl := tt.k8sClient(t)
			h := NewTektonTriggerTemplateManager(k8sCl)
			err := h.CreatePipelineRun(
				context.Background(),
				tt.args.ns,
				tt.args.cdStageDeployName,
				tt.args.rawPipeRun,
				tt.args.appPayload,
				tt.args.stage,
				tt.args.pipeline,
				tt.args.clusterSecret,
			)

			tt.wantErr(t, err)
			tt.want(t, k8sCl)
		})
	}
}

func TestTektonTriggerTemplateManager_CreatePendingPipelineRun(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, tektonpipelineApi.AddToScheme(scheme))
	require.NoError(t, tektonTriggersApi.AddToScheme(scheme))

	type args struct {
		ns                string
		cdStageDeployName string
		rawPipeRun        []byte
		appPayload        []byte
		stage             []byte
		pipeline          []byte
		clusterSecret     []byte
	}

	tests := []struct {
		name      string
		args      args
		k8sClient func(t *testing.T) client.Client
		wantErr   require.ErrorAssertionFunc
		want      func(t *testing.T, k8sCl client.Client)
	}{
		{
			name: "create pipeline run successfully",
			args: args{
				ns:                "default",
				cdStageDeployName: "deploy-app",
				rawPipeRun:        pipelineRunTemplate,
				appPayload:        []byte(`{"app1":"1.1", "app2":"2.0"}`),
				stage:             []byte("dev"),
				pipeline:          []byte("pipeline-1"),
				clusterSecret:     []byte("cl-secret"),
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				l := &tektonpipelineApi.PipelineRunList{}
				require.NoError(t, k8sCl.List(context.Background(), l, client.InNamespace("default")))
				require.Len(t, l.Items, 1)

				run := l.Items[0]

				require.Equal(t, tektonpipelineApi.PipelineRunSpecStatusPending, string(run.Spec.Status))
				require.Len(t, run.Spec.Params, 4)

				for _, p := range run.Spec.Params {
					switch p.Name {
					case "APPLICATIONS_PAYLOAD":
						require.Equal(t, `{"app1":"1.1", "app2":"2.0"}`, p.Value.StringVal)
					case "CDSTAGE":
						require.Equal(t, "dev", p.Value.StringVal)
					case "CDPIPELINE":
						require.Equal(t, "pipeline-1", p.Value.StringVal)
					case "KUBECONFIG_SECRET_NAME":
						require.Equal(t, "cl-secret", p.Value.StringVal)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sCl := tt.k8sClient(t)
			h := NewTektonTriggerTemplateManager(k8sCl)
			err := h.CreatePendingPipelineRun(
				context.Background(),
				tt.args.ns,
				tt.args.cdStageDeployName,
				tt.args.rawPipeRun,
				tt.args.appPayload,
				tt.args.stage,
				tt.args.pipeline,
				tt.args.clusterSecret,
			)

			tt.wantErr(t, err)
			tt.want(t, k8sCl)
		})
	}
}
