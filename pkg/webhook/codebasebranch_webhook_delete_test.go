package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func deleteWebhookScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	return scheme
}

func deleteWebhookBranch() *codebaseApi.CodebaseBranch {
	return &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{Name: "app-feature", Namespace: "default"},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "app",
			BranchName:   "feature",
		},
	}
}

func TestCodebaseBranchValidationWebhook_ValidateDelete_BranchInUse(t *testing.T) {
	tests := []struct {
		name    string
		objects []runtime.Object
		wantErr string
	}{
		{
			name: "rejects deletion of branch used by CDPipeline",
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
				},
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
					Spec:       pipelineApi.CDPipelineSpec{InputDockerStreams: []string{"app-feature"}},
				},
			},
			wantErr: "used by CDPipeline demo",
		},
		{
			name: "allows deletion of unused branch",
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
				},
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
					Spec:       pipelineApi.CDPipelineSpec{InputDockerStreams: []string{"other-main"}},
				},
			},
		},
		{
			name: "allows deletion when owning codebase is terminating",
			objects: []runtime.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "app",
						Namespace:         "default",
						DeletionTimestamp: ptr.To(metav1.Now()),
						Finalizers:        []string{"keep"},
					},
				},
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
					Spec:       pipelineApi.CDPipelineSpec{InputDockerStreams: []string{"app-feature"}},
				},
			},
		},
		{
			name: "allows deletion when owning codebase is gone",
			objects: []runtime.Object{
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
					Spec:       pipelineApi.CDPipelineSpec{InputDockerStreams: []string{"app-feature"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(deleteWebhookScheme(t)).
				WithRuntimeObjects(tt.objects...).
				Build()

			w := NewCodebaseBranchValidationWebhook(k8sClient, ctrl.Log)

			_, err := w.ValidateDelete(context.Background(), deleteWebhookBranch())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
