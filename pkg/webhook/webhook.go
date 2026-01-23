package webhook

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// RegisterValidationWebHook registers a new webhook for validating CRD.
func RegisterValidationWebHook(ctx context.Context, mgr ctrl.Manager, namespace string) error {
	// mgr.GetAPIReader() is used to read objects before cache is started.
	certService := NewCertService(mgr.GetAPIReader(), mgr.GetClient())
	if err := certService.PopulateCertificates(ctx, namespace); err != nil {
		return fmt.Errorf("failed to populate certificates: %w", err)
	}

	if err := NewCodebaseValidationWebhook(
		mgr.GetClient(),
		ctrl.Log.WithName("codebase-webhook"),
	).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := NewCodebaseBranchValidationWebhook(mgr.GetClient(), ctrl.Log).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	return (&ProtectedLabelValidationWebhook{}).SetupWebhookWithManager(
		mgr,
		&codebaseApi.CodebaseImageStream{},
		&codebaseApi.GitServer{},
	)
}
