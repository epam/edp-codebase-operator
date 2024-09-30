package chain

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutCDStageDeploy_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineAPi.AddToScheme(scheme))

	tests := []struct {
		name        string
		imageStream *codebaseApi.CodebaseImageStream
		client      func(t *testing.T) client.Client
		wantErr     require.ErrorAssertionFunc
		want        func(t *testing.T, k8scl client.Client)
	}{
		{
			name: "successfully created CDStageDeploy",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: time.Now().Format(time.RFC3339)}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "ci-dev",
								Namespace: "default",
							},
							Spec: pipelineAPi.StageSpec{
								TriggerType: pipelineAPi.TriggerTypeAutoDeploy,
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-prod",
								},
							},
						},
					).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8scl client.Client) {
				cdStageDeploys := &codebaseApi.CDStageDeployList{}
				require.NoError(t,
					k8scl.List(
						context.Background(),
						cdStageDeploys,
						client.InNamespace("default"),
						client.MatchingLabels{
							codebaseApi.CdPipelineLabel: "ci",
							codebaseApi.CdStageLabel:    "ci-dev",
						}))
				require.Len(t, cdStageDeploys.Items, 1)
			},
		},
		{
			name: "successfully created CDStageDeploy if one CDStageDeploy already exists",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: time.Now().Format(time.RFC3339)}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "ci-dev",
								Namespace: "default",
							},
							Spec: pipelineAPi.StageSpec{
								TriggerType: pipelineAPi.TriggerTypeAutoDeploy,
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-dev",
								},
							},
						},
					).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8scl client.Client) {
				cdStageDeploys := &codebaseApi.CDStageDeployList{}
				require.NoError(t,
					k8scl.List(
						context.Background(),
						cdStageDeploys,
						client.InNamespace("default"),
						client.MatchingLabels{
							codebaseApi.CdPipelineLabel: "ci",
							codebaseApi.CdStageLabel:    "ci-dev",
						}))
				require.Len(t, cdStageDeploys.Items, 2)
			},
		},
		{
			name: "skip CDStageDeploy creation if more than one CDStageDeploy already exists",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: time.Now().Format(time.RFC3339)}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "ci-dev",
								Namespace: "default",
							},
							Spec: pipelineAPi.StageSpec{
								TriggerType: pipelineAPi.TriggerTypeAutoDeploy,
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy1",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-dev",
								},
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy2",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-dev",
								},
							},
						},
					).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8scl client.Client) {
				cdStageDeploys := &codebaseApi.CDStageDeployList{}
				require.NoError(t,
					k8scl.List(
						context.Background(),
						cdStageDeploys,
						client.InNamespace("default"),
						client.MatchingLabels{
							codebaseApi.CdPipelineLabel: "ci",
							codebaseApi.CdStageLabel:    "ci-dev",
						}))
				require.Len(t, cdStageDeploys.Items, 2)
			},
		},
		{
			name: "don't skip CDStageDeploy creation if more than one CDStageDeploy already exists for AutoStable trigger type",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: time.Now().Format(time.RFC3339)}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "ci-dev",
								Namespace: "default",
							},
							Spec: pipelineAPi.StageSpec{
								TriggerType: pipelineAPi.TriggerTypeAutoStable,
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy1",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-dev",
								},
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy2",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-dev",
								},
							},
						},
					).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8scl client.Client) {
				cdStageDeploys := &codebaseApi.CDStageDeployList{}
				require.NoError(t,
					k8scl.List(
						context.Background(),
						cdStageDeploys,
						client.InNamespace("default"),
						client.MatchingLabels{
							codebaseApi.CdPipelineLabel: "ci",
							codebaseApi.CdStageLabel:    "ci-dev",
						}))
				require.Len(t, cdStageDeploys.Items, 3)
			},
		},
		{
			name: "failed to create CDStageDeploy - CDStage not found",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: time.Now().Format(time.RFC3339)}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get CDStage")
			},
			want: func(t *testing.T, k8scl client.Client) {},
		},
		{
			name: "failed to create CDStageDeploy - invalid tag",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: "invalid"}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "ci-dev",
								Namespace: "default",
							},
							Spec: pipelineAPi.StageSpec{
								TriggerType: pipelineAPi.TriggerTypeAutoDeploy,
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get last tag")
			},
			want: func(t *testing.T, k8scl client.Client) {},
		},
		{
			name: "failed to create CDStageDeploy - invalid env label",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"invalid": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "label must be in format cd-pipeline-name/stage-name")
			},
			want: func(t *testing.T, k8scl client.Client) {},
		},
		{
			name: "failed to create CDStageDeploy - no tags",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tags are not defined in spec")
			},
			want: func(t *testing.T, k8scl client.Client) {},
		},
		{
			name: "failed to create CDStageDeploy - no codebase",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
					Labels: map[string]string{
						"ci/dev": "",
					},
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "",
					ImageName: "latest",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "codebase is not defined in spec")
			},
			want: func(t *testing.T, k8scl client.Client) {},
		},
		{
			name: "no env labels - skip CDStageDeploy creation",
			imageStream: &codebaseApi.CodebaseImageStream{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseImageStreamSpec{
					Codebase:  "app",
					ImageName: "latest",
					Tags:      []codebaseApi.Tag{{Name: "latest", Created: time.Now().Format(time.RFC3339)}},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&pipelineAPi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "ci-dev",
								Namespace: "default",
							},
						},
						&codebaseApi.CDStageDeploy{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "test-stage-deploy",
								Namespace: "default",
								Labels: map[string]string{
									codebaseApi.CdPipelineLabel: "ci",
									codebaseApi.CdStageLabel:    "ci-prod",
								},
							},
						},
					).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8scl client.Client) {
				cdStageDeploys := &codebaseApi.CDStageDeployList{}
				require.NoError(t,
					k8scl.List(
						context.Background(),
						cdStageDeploys,
						client.InNamespace("default"),
						client.MatchingLabels{
							codebaseApi.CdPipelineLabel: "ci",
							codebaseApi.CdStageLabel:    "ci-dev",
						}))
				require.Len(t, cdStageDeploys.Items, 0)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := PutCDStageDeploy{
				client: tt.client(t),
			}
			tt.wantErr(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.imageStream))
			tt.want(t, h.client)
		})
	}
}
