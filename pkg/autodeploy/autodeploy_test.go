package autodeploy

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestStrategyManager_GetAppPayloadForAllLatestStrategy(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name      string
		pipeline  *pipelineAPi.CDPipeline
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "get payload successfully",
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main", "app2-feature/2"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
								},
								{
									Name:    "1.3",
									Created: time.Now().Add(time.Hour * 2).Format(time.RFC3339),
								},
								{
									Name:    "1.1",
									Created: time.Now().Add(time.Hour).Format(time.RFC3339),
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-feature-2",
							Namespace: "default",
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"1.3"},"app2":{"imageTag":"1.0"}}`,
			wantErr: require.NoError,
		},
		{
			name: "latest tag not found",
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
						},
					},
				).Build()
			},
			want: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "last tag not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewStrategyManager(tt.k8sClient(t))
			got, err := h.GetAppPayloadForAllLatestStrategy(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.pipeline)

			tt.wantErr(t, err)
			if tt.want != "" {
				assert.JSONEq(t, tt.want, string(got))
			}
		})
	}
}

func TestStrategyManager_GetAppPayloadForCurrentWithStableStrategy(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineAPi.AddToScheme(scheme))

	tests := []struct {
		name      string
		current   codebaseApi.CodebaseTag
		pipeline  *pipelineAPi.CDPipeline
		stage     *pipelineAPi.Stage
		k8sClient func(t *testing.T) client.Client
		want      string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "get payload successfully",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "1.1",
			},
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main", "app2-main", "app3-main"},
					Applications:       []string{"app1", "app2", "app3"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
								},
								{
									Name:    "1.3",
									Created: time.Now().Add(time.Hour * 2).Format(time.RFC3339),
								},
								{
									Name:    "1.1",
									Created: time.Now().Add(time.Hour).Format(time.RFC3339),
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-main",
							Namespace: "default",
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
								},
								{
									Name:    "1.3",
									Created: time.Now().Add(time.Hour * 2).Format(time.RFC3339),
								},
								{
									Name:    "1.1",
									Created: time.Now().Add(time.Hour).Format(time.RFC3339),
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app3-main",
							Namespace: "default",
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app3",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
								},
								{
									Name:    "1.3",
									Created: time.Now().Add(time.Hour * 2).Format(time.RFC3339),
								},
								{
									Name:    "1.1",
									Created: time.Now().Add(time.Hour).Format(time.RFC3339),
								},
							},
						},
					},
				).Build()
			},
			stage: &pipelineAPi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.edp.epam.com/app3": "1.0",
					},
				},
			},
			want:    `{"app1":{"imageTag":"1.1"},"app2":{"imageTag":"1.3"},"app3":{"imageTag":"1.0"}}`,
			wantErr: require.NoError,
		},
		{
			name: "codebaseimagestream not found",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "1.1",
			},
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main"},
					Applications:       []string{"app1"},
				},
			},
			stage: &pipelineAPi.Stage{},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
			},
			want: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get app1-main CodebaseImageStream")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewStrategyManager(tt.k8sClient(t))
			got, err := h.GetAppPayloadForCurrentWithStableStrategy(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.current,
				tt.pipeline,
				tt.stage,
			)

			tt.wantErr(t, err)
			if tt.want != "" {
				assert.JSONEq(t, tt.want, string(got))
			}
		})
	}
}
