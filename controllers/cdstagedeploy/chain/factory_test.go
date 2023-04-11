package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestCreateDefChain(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		deploy  *codebaseApi.CDStageDeploy
		objects []runtime.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should create jenkins chain",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Tag: codebaseApi.CodebaseTag{
						Codebase: "app1",
					},
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "app1",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseSpec{
						CiTool: util.CIJenkins,
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "should create tekton chain",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Tag: codebaseApi.CodebaseTag{
						Codebase: "app1",
					},
				},
			},
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "app1",
						Namespace: "default",
					},
					Spec: codebaseApi.CodebaseSpec{
						CiTool: util.CITekton,
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to get codebase",
			deploy: &codebaseApi.CDStageDeploy{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: codebaseApi.CDStageDeploySpec{
					Tag: codebaseApi.CodebaseTag{
						Codebase: "app1",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)

				assert.Contains(t, err.Error(), "failed to get codebase")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := CreateDefChain(
				context.Background(),
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				tt.deploy,
			)

			tt.wantErr(t, err)
		})
	}
}
