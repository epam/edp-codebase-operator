package webhook

import (
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// RegisterValidationWebHook registers a new webhook for validating CRD.
func RegisterValidationWebHook(mgr ctrl.Manager) error {
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
