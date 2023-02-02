package webhook

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterValidationWebHook registers a new webhook for validating CRD.
func RegisterValidationWebHook(ctx context.Context, mgr ctrl.Manager, namespace string) error {
	// mgr.GetAPIReader() is used to read objects before cache is started.
	certService := NewCertService(mgr.GetAPIReader(), mgr.GetClient())
	if err := certService.PopulateCertificates(ctx, namespace); err != nil {
		return fmt.Errorf("failed to populate certificates: %w", err)
	}

	codebaseWebHook := NewCodebaseValidationWebhook(mgr.GetClient(), ctrl.Log.WithName("codebase-webhook"))
	if err := codebaseWebHook.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}
