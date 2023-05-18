package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/argocd"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestUpdateArgoApplicationTag_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		deploy  *codebaseApi.CDStageDeploy
		objects []client.Object
		wantErr require.ErrorAssertionFunc
		assert  func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader)
	}{
		{
			name: "should update argo application tag",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "test-pipeline",
					Stage:    "test-stage",
					Tag: codebaseApi.CodebaseTag{
						Codebase: "test-codebase",
						Tag:      "2.0.0",
					},
				},
			},
			objects: []client.Object{
				argocd.NewArgoCDApplication(
					argocd.WithName("test"),
					argocd.WithNamespace("default"),
					argocd.WithLabels(map[string]string{
						"app.edp.epam.com/app-name": "test-codebase",
						"app.edp.epam.com/pipeline": "test-pipeline",
						"app.edp.epam.com/stage":    "test-stage",
					}),
					argocd.WithSpec(map[string]interface{}{
						"source": map[string]interface{}{
							"targetRevision": "1.0.0",
							"helm": map[string]interface{}{
								"parameters": []interface{}{
									map[string]interface{}{
										"name":  "image.tag",
										"value": "1.0.0",
									},
									map[string]interface{}{
										"name":  "image.repository",
										"value": "test-repo",
									},
								},
							},
						},
					}),
				),
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
				},
			},
			wantErr: require.NoError,
			assert: func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader) {
				app := argocd.NewArgoCDApplication()
				require.NoError(t, reader.Get(context.Background(), client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				}, app))

				var patch argoApplicationImagePatch
				require.NoError(t, mapstructure.Decode(app.Object, &patch))

				assert.Equal(t, "2.0.0", patch.Spec.Source.TargetRevision)
				require.Len(t, patch.Spec.Source.Helm.Parameters, 2)
				assert.Equal(t, "2.0.0", patch.Spec.Source.Helm.Parameters[0].Value)
			},
		},
		{
			name: "should update argo application tag with edp versioning",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "test-pipeline",
					Stage:    "test-stage",
					Tag: codebaseApi.CodebaseTag{
						Codebase: "test-codebase",
						Tag:      "2.0.0",
					},
				},
			},
			objects: []client.Object{
				argocd.NewArgoCDApplication(
					argocd.WithName("test"),
					argocd.WithNamespace("default"),
					argocd.WithLabels(map[string]string{
						"app.edp.epam.com/app-name": "test-codebase",
						"app.edp.epam.com/pipeline": "test-pipeline",
						"app.edp.epam.com/stage":    "test-stage",
					}),
					argocd.WithSpec(map[string]interface{}{
						"source": map[string]interface{}{
							"targetRevision": "1.0.0",
							"helm": map[string]interface{}{
								"parameters": []interface{}{
									map[string]interface{}{
										"name":  "image.tag",
										"value": "1.0.0",
									},
									map[string]interface{}{
										"name":  "image.repository",
										"value": "test-repo",
									},
								},
							},
						},
					}),
				),
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseSpec{
						Versioning: codebaseApi.Versioning{
							Type:      util.VersioningTypeEDP,
							StartFrom: pointer.String("2.0.0"),
						},
					},
				},
			},
			wantErr: require.NoError,
			assert: func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader) {
				app := argocd.NewArgoCDApplication()
				require.NoError(t, reader.Get(context.Background(), client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				}, app))

				var patch argoApplicationImagePatch
				require.NoError(t, mapstructure.Decode(app.Object, &patch))

				assert.Equal(t, "build/2.0.0", patch.Spec.Source.TargetRevision)
				require.Len(t, patch.Spec.Source.Helm.Parameters, 2)
				assert.Equal(t, "2.0.0", patch.Spec.Source.Helm.Parameters[0].Value)
			},
		},
		{
			name: "failed to get codebase",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "test-pipeline",
					Stage:    "test-stage",
					Tag: codebaseApi.CodebaseTag{
						Codebase: "test-codebase",
						Tag:      "2.0.0",
					},
				},
			},
			objects: []client.Object{
				argocd.NewArgoCDApplication(
					argocd.WithName("test"),
					argocd.WithNamespace("default"),
					argocd.WithLabels(map[string]string{
						"app.edp.epam.com/app-name": "test-codebase",
						"app.edp.epam.com/pipeline": "test-pipeline",
						"app.edp.epam.com/stage":    "test-stage",
					}),
					argocd.WithSpec(map[string]interface{}{
						"source": map[string]interface{}{
							"targetRevision": "1.0.0",
							"helm": map[string]interface{}{
								"parameters": []interface{}{
									map[string]interface{}{
										"name":  "image.tag",
										"value": "1.0.0",
									},
									map[string]interface{}{
										"name":  "image.repository",
										"value": "test-repo",
									},
								},
							},
						},
					}),
				),
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get codebase")
			},
			assert: func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader) {
			},
		},
		{
			name: "argo application has empty spec",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "test-pipeline",
					Stage:    "test-stage",
					Tag: codebaseApi.CodebaseTag{
						Codebase: "test-codebase",
						Tag:      "2.0.0",
					},
				},
			},
			objects: []client.Object{
				argocd.NewArgoCDApplication(
					argocd.WithName("test"),
					argocd.WithNamespace("default"),
					argocd.WithLabels(map[string]string{
						"app.edp.epam.com/app-name": "test-codebase",
						"app.edp.epam.com/pipeline": "test-pipeline",
						"app.edp.epam.com/stage":    "test-stage",
					}),
				),
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
				},
			},
			wantErr: require.NoError,
			assert: func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader) {
				app := argocd.NewArgoCDApplication()
				require.NoError(t, reader.Get(context.Background(), client.ObjectKey{
					Name:      "test",
					Namespace: "default",
				}, app))

				var patch argoApplicationImagePatch
				require.NoError(t, mapstructure.Decode(app.Object, &patch))

				assert.Equal(t, "2.0.0", patch.Spec.Source.TargetRevision)
			},
		},
		{
			name: "failed with multiple argo applications",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "test-pipeline",
					Stage:    "test-stage",
					Tag: codebaseApi.CodebaseTag{
						Codebase: "test-codebase",
						Tag:      "2.0.0",
					},
				},
			},
			objects: []client.Object{
				argocd.NewArgoCDApplication(
					argocd.WithName("app1"),
					argocd.WithNamespace("default"),
					argocd.WithLabels(map[string]string{
						"app.edp.epam.com/app-name": "test-codebase",
						"app.edp.epam.com/pipeline": "test-pipeline",
						"app.edp.epam.com/stage":    "test-stage",
					}),
				),
				argocd.NewArgoCDApplication(
					argocd.WithName("app2"),
					argocd.WithNamespace("default"),
					argocd.WithLabels(map[string]string{
						"app.edp.epam.com/app-name": "test-codebase",
						"app.edp.epam.com/pipeline": "test-pipeline",
						"app.edp.epam.com/stage":    "test-stage",
					}),
				),
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "found multiple Argo Application with the provided labels")
			},
			assert: func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader) {},
		},
		{
			name: "argo application not found",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "test-pipeline",
					Stage:    "test-stage",
					Tag: codebaseApi.CodebaseTag{
						Codebase: "test-codebase",
						Tag:      "2.0.0",
					},
				},
			},
			wantErr: require.NoError,
			assert:  func(t *testing.T, deploy *codebaseApi.CDStageDeploy, reader client.Reader) {},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			h := &UpdateArgoApplicationTag{
				client: cl,
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.deploy)

			tt.wantErr(t, err)
			tt.assert(t, tt.deploy, cl)
		})
	}
}
