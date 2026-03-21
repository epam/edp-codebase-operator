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
					InputDockerStreams: []string{"app1-main", "app2-feature-2"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-feature-2",
							},
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
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
		{
			name: "payload includes imageDigest when CBIS tag has digest",
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:aaa111",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"1.0","imageDigest":"sha256:aaa111"}}`,
			wantErr: require.NoError,
		},
		{
			name: "payload omits imageDigest when CBIS tag has no digest",
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
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
			want:    `{"app1":{"imageTag":"1.0"}}`,
			wantErr: require.NoError,
		},
		{
			name: "mixed digest: some tags with digest, some without",
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main", "app2-main"},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "2.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:abc123",
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "3.0",
									Created: time.Now().Format(time.RFC3339),
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"2.0","imageDigest":"sha256:abc123"},"app2":{"imageTag":"3.0"}}`,
			wantErr: require.NoError,
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-main",
							},
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
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app3-main",
							},
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
				require.Contains(t, err.Error(), "failed to get CodebaseImageStream")
			},
		},
		{
			name: "current app with digest from CDStageDeploy",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "2.0",
				Digest:   "sha256:current111",
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
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "2.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:current111",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"2.0","imageDigest":"sha256:current111"}}`,
			wantErr: require.NoError,
		},
		{
			name: "current app without digest in CBIS",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "2.0",
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
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "2.0",
									Created: time.Now().Format(time.RFC3339),
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"2.0"}}`,
			wantErr: require.NoError,
		},
		{
			name: "current app without digest in CDStageDeploy, backfilled from CBIS",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "2.0",
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
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "2.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:backfilled",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"2.0","imageDigest":"sha256:backfilled"}}`,
			wantErr: require.NoError,
		},
		{
			name: "stable app from annotation with digest from CBIS",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "1.0",
			},
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main", "app2-main"},
					Applications:       []string{"app1", "app2"},
				},
			},
			stage: &pipelineAPi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.edp.epam.com/app2": "3.0",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:app1digest",
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "3.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:app2digest",
								},
								{
									Name:    "4.0",
									Created: time.Now().Add(time.Hour).Format(time.RFC3339),
									Digest:  "sha256:app2latest",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"1.0","imageDigest":"sha256:app1digest"},"app2":{"imageTag":"3.0","imageDigest":"sha256:app2digest"}}`,
			wantErr: require.NoError,
		},
		{
			name: "stable app annotation tag not found in CBIS",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "1.0",
			},
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main", "app2-main"},
					Applications:       []string{"app1", "app2"},
				},
			},
			stage: &pipelineAPi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.edp.epam.com/app2": "old-tag",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:app1digest",
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "3.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:app2digest",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"1.0","imageDigest":"sha256:app1digest"},"app2":{"imageTag":"old-tag"}}`,
			wantErr: require.NoError,
		},
		{
			name: "latest fallback app with digest",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "1.0",
			},
			pipeline: &pipelineAPi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: pipelineAPi.CDPipelineSpec{
					InputDockerStreams: []string{"app1-main", "app2-main"},
					Applications:       []string{"app1", "app2"},
				},
			},
			stage: &pipelineAPi.Stage{},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:app1digest",
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "5.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:app2latest",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"1.0","imageDigest":"sha256:app1digest"},"app2":{"imageTag":"5.0","imageDigest":"sha256:app2latest"}}`,
			wantErr: require.NoError,
		},
		{
			name: "mixed: current with digest, stable without, latest with",
			current: codebaseApi.CodebaseTag{
				Codebase: "app1",
				Tag:      "1.0",
				Digest:   "sha256:currentdigest",
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
			stage: &pipelineAPi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.edp.epam.com/app2": "2.0",
					},
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app1-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app1",
							Tags: []codebaseApi.Tag{
								{
									Name:    "1.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:currentdigest",
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app2-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app2",
							Tags: []codebaseApi.Tag{
								{
									Name:    "2.0",
									Created: time.Now().Format(time.RFC3339),
								},
							},
						},
					},
					&codebaseApi.CodebaseImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app3-main",
							Namespace: "default",
							Labels: map[string]string{
								codebaseApi.CodebaseBranchLabel: "app3-main",
							},
						},
						Spec: codebaseApi.CodebaseImageStreamSpec{
							Codebase: "app3",
							Tags: []codebaseApi.Tag{
								{
									Name:    "7.0",
									Created: time.Now().Format(time.RFC3339),
									Digest:  "sha256:latestdigest",
								},
							},
						},
					},
				).Build()
			},
			want:    `{"app1":{"imageTag":"1.0","imageDigest":"sha256:currentdigest"},"app2":{"imageTag":"2.0"},"app3":{"imageTag":"7.0","imageDigest":"sha256:latestdigest"}}`,
			wantErr: require.NoError,
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

func TestStrategyManager_getLatestTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		imageStream *codebaseApi.CodebaseImageStream
		wantCb      string
		wantTag     codebaseApi.Tag
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "returns full Tag struct with digest",
			imageStream: &codebaseApi.CodebaseImageStream{
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase: "myapp",
					Tags: []codebaseApi.Tag{
						{
							Name:    "1.0",
							Created: time.Now().Format(time.RFC3339),
							Digest:  "sha256:abc123",
						},
						{
							Name:    "2.0",
							Created: time.Now().Add(time.Hour).Format(time.RFC3339),
							Digest:  "sha256:def456",
						},
					},
				},
			},
			wantCb: "myapp",
			wantTag: codebaseApi.Tag{
				Name:   "2.0",
				Digest: "sha256:def456",
			},
			wantErr: require.NoError,
		},
		{
			name: "returns error when no tags",
			imageStream: &codebaseApi.CodebaseImageStream{
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase: "myapp",
				},
			},
			wantCb:  "",
			wantTag: codebaseApi.Tag{},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "last tag not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &StrategyManager{}
			codebase, tag, err := h.getLatestTag(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.imageStream,
			)

			tt.wantErr(t, err)

			if err == nil {
				assert.Equal(t, tt.wantCb, codebase)
				assert.Equal(t, tt.wantTag.Name, tag.Name)
				assert.Equal(t, tt.wantTag.Digest, tag.Digest)
			}
		})
	}
}

func Test_findDigestByTagName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cbis    *codebaseApi.CodebaseImageStream
		tagName string
		want    string
	}{
		{
			name:    "nil CBIS returns empty string",
			cbis:    nil,
			tagName: "1.0",
			want:    "",
		},
		{
			name: "tag found with digest",
			cbis: &codebaseApi.CodebaseImageStream{
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Tags: []codebaseApi.Tag{
						{
							Name:   "1.0",
							Digest: "sha256:found",
						},
					},
				},
			},
			tagName: "1.0",
			want:    "sha256:found",
		},
		{
			name: "tag found without digest",
			cbis: &codebaseApi.CodebaseImageStream{
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Tags: []codebaseApi.Tag{
						{
							Name: "1.0",
						},
					},
				},
			},
			tagName: "1.0",
			want:    "",
		},
		{
			name: "tag not found",
			cbis: &codebaseApi.CodebaseImageStream{
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Tags: []codebaseApi.Tag{
						{
							Name:   "2.0",
							Digest: "sha256:other",
						},
					},
				},
			},
			tagName: "1.0",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDigestByTagName(tt.cbis, tt.tagName)
			assert.Equal(t, tt.want, got)
		})
	}
}
