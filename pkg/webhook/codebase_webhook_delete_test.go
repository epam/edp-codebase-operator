package webhook

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	admissionv1 "k8s.io/api/admission/v1"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebase"
)

// codebaseDeleteClientBuilder mirrors the production wiring: the Codebase deletion check
// selects branches on BranchCodebaseNameIndex, which a fake client only serves when the
// index is registered on the builder.
func codebaseDeleteClientBuilder(t *testing.T) *fake.ClientBuilder {
	t.Helper()

	return fake.NewClientBuilder().
		WithScheme(deleteWebhookScheme(t)).
		WithIndex(&codebaseApi.CodebaseBranch{}, codebase.BranchCodebaseNameIndex, codebase.IndexBranchByCodebaseName)
}

func deleteWebhookCodebase() *codebaseApi.Codebase {
	return &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
	}
}

func codebaseDeleteAdmissionCtx() context.Context {
	return admission.NewContextWithRequest(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name:      "app",
			Namespace: "default",
		},
	})
}

func TestCodebaseValidationWebhook_ValidateDelete_CodebaseInUse(t *testing.T) {
	tests := []struct {
		name    string
		objects []runtime.Object
		wantErr string
	}{
		{
			name: "rejects deletion of application component used by CDPipeline",
			objects: []runtime.Object{
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
					Spec:       pipelineApi.CDPipelineSpec{Applications: []string{"app"}},
				},
			},
			wantErr: "used by CDPipeline demo",
		},
		{
			name: "rejects deletion of autotest component used by a Stage quality gate",
			objects: []runtime.Object{
				&pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
					Spec: pipelineApi.StageSpec{
						CdPipeline: "demo",
						QualityGates: []pipelineApi.QualityGate{{
							QualityGateType: "autotests",
							AutotestName:    ptr.To("app"),
							BranchName:      ptr.To("main"),
						}},
					},
				},
			},
			wantErr: "used by Stage demo-dev of CDPipeline demo",
		},
		{
			// A manual quality gate leaves AutotestName unset and must never be
			// mistaken for a match.
			name: "allows deletion when only a manual quality gate exists",
			objects: []runtime.Object{
				&pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
					Spec: pipelineApi.StageSpec{
						CdPipeline: "demo",
						QualityGates: []pipelineApi.QualityGate{{
							QualityGateType: "manual",
							StepName:        "approve",
						}},
					},
				},
			},
		},
		{
			name: "allows deletion of unused codebase",
			objects: []runtime.Object{
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
					Spec:       pipelineApi.CDPipelineSpec{Applications: []string{"other-app"}},
				},
			},
		},
		{
			name: "allows deletion when referencing CDPipeline is terminating",
			objects: []runtime.Object{
				&pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "demo",
						Namespace:         "default",
						DeletionTimestamp: ptr.To(metav1.Now()),
						Finalizers:        []string{"keep"},
					},
					Spec: pipelineApi.CDPipelineSpec{Applications: []string{"app"}},
				},
			},
		},
		{
			name:    "allows deletion when CD pipeline CRDs are not installed",
			objects: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := codebaseDeleteClientBuilder(t).
				WithRuntimeObjects(tt.objects...).
				Build()

			w := NewCodebaseValidationWebhook(k8sClient, ctrl.Log)

			_, err := w.ValidateDelete(codebaseDeleteAdmissionCtx(), deleteWebhookCodebase())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCodebaseValidationWebhook_ValidateDelete_StatusError(t *testing.T) {
	objects := []runtime.Object{
		&pipelineApi.CDPipeline{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
			Spec:       pipelineApi.CDPipelineSpec{Applications: []string{"app"}},
		},
	}

	k8sClient := codebaseDeleteClientBuilder(t).
		WithRuntimeObjects(objects...).
		Build()

	w := NewCodebaseValidationWebhook(k8sClient, ctrl.Log)

	_, err := w.ValidateDelete(codebaseDeleteAdmissionCtx(), deleteWebhookCodebase())
	require.Error(t, err)

	var statusErr *apierrors.StatusError
	require.True(t, errors.As(err, &statusErr))

	status := statusErr.Status()
	assert.Equal(t, int32(http.StatusForbidden), status.Code)
	assert.Equal(t, metav1.StatusReasonForbidden, status.Reason)
	assert.Equal(t,
		"Codebase app cannot be deleted because it is used by CDPipeline demo (applications); "+
			"remove it from the deployment first",
		status.Message,
	)
	require.NotNil(t, status.Details)
	require.Len(t, status.Details.Causes, 1)
	assert.Equal(t, metav1.CauseTypeForbidden, status.Details.Causes[0].Type)
	assert.Equal(t, "CDPipeline demo (applications)", status.Details.Causes[0].Message)
	// Field is the machine-readable half of the cause; clients key off it instead of
	// parsing Message.
	assert.Equal(t, "spec.applications", status.Details.Causes[0].Field)
}

// Every blocking reference is reported at once, so the user sees the full set to clean up
// rather than discovering them one delete at a time.
func TestCodebaseValidationWebhook_ValidateDelete_StatusErrorMultipleReferences(t *testing.T) {
	objects := []runtime.Object{
		&pipelineApi.CDPipeline{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
			Spec:       pipelineApi.CDPipelineSpec{Applications: []string{"app"}},
		},
		&pipelineApi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "demo-dev", Namespace: "default"},
			Spec: pipelineApi.StageSpec{
				CdPipeline: "demo",
				QualityGates: []pipelineApi.QualityGate{{
					QualityGateType: "autotests",
					AutotestName:    ptr.To("app"),
					BranchName:      ptr.To("main"),
				}},
			},
		},
	}

	k8sClient := codebaseDeleteClientBuilder(t).
		WithRuntimeObjects(objects...).
		Build()

	w := NewCodebaseValidationWebhook(k8sClient, ctrl.Log)

	_, err := w.ValidateDelete(codebaseDeleteAdmissionCtx(), deleteWebhookCodebase())
	require.Error(t, err)

	var statusErr *apierrors.StatusError
	require.True(t, errors.As(err, &statusErr))

	status := statusErr.Status()
	assert.Equal(t,
		"Codebase app cannot be deleted because it is used by CDPipeline demo (applications); "+
			"Stage demo-dev of CDPipeline demo (autotest quality gate); "+
			"remove it from the deployment first",
		status.Message,
	)
	require.NotNil(t, status.Details)
	require.Len(t, status.Details.Causes, 2)
	assert.Equal(t, "spec.applications", status.Details.Causes[0].Field)
	assert.Equal(t, "spec.qualityGates.autotestName", status.Details.Causes[1].Field)
}

// An object of another kind cannot be checked for usage; validation is skipped rather
// than denied, since the request is not one this webhook governs.
func TestCodebaseValidationWebhook_ValidateDelete_WrongObjectType(t *testing.T) {
	w := NewCodebaseValidationWebhook(codebaseDeleteClientBuilder(t).Build(), ctrl.Log)

	warnings, err := w.ValidateDelete(codebaseDeleteAdmissionCtx(), &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{Name: "app-main", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.Nil(t, warnings)
}

// When usage cannot be determined the deletion must be refused: treating an unverifiable
// codebase as unused is what this webhook exists to prevent.
func TestCodebaseValidationWebhook_ValidateDelete_UsageCheckErrorDenies(t *testing.T) {
	k8sClient := codebaseDeleteClientBuilder(t).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return apierrors.NewForbidden(
					schema.GroupResource{Group: "v2.edp.epam.com", Resource: "cdpipelines"}, "", errors.New("no access"))
			},
		}).
		Build()

	w := NewCodebaseValidationWebhook(k8sClient, ctrl.Log)

	_, err := w.ValidateDelete(codebaseDeleteAdmissionCtx(), deleteWebhookCodebase())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check Codebase usage")
}
