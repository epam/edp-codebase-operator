package webhook

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/epam/edp-codebase-operator/v2/api/v1"
)

const listLimit = 1000

//+kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-codebase,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=codebases,verbs=create;update,versions=v1,name=vcodebase.kb.io,admissionReviewVersions=v1

// CodebaseValidationWebhook is a webhook for validating Codebase CRD.
type CodebaseValidationWebhook struct {
	client client.Client
	log    logr.Logger
}

// NewCodebaseValidationWebhook creates a new webhook for validating Codebase CR.
func NewCodebaseValidationWebhook(k8sClient client.Client, log logr.Logger) *CodebaseValidationWebhook {
	return &CodebaseValidationWebhook{client: k8sClient, log: log.WithName("codebase-webhook")}
}

// SetupWebhookWithManager sets up the webhook with the manager for Codebase CR.
func (r *CodebaseValidationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&v1.Codebase{}).
		WithValidator(r).
		Complete()

	if err != nil {
		return fmt.Errorf("failed to build Codebase validation webhook: %w", err)
	}

	return nil
}

var _ webhook.CustomValidator = &CodebaseValidationWebhook{}

// ValidateCreate is a webhook for validating the creation of the Codebase CR.
func (r *CodebaseValidationWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Errorf("expected admission.Request in ctx: %w", err).Error())
	}

	r.log.Info("validate create", "name", req.Name)

	createdCodebase, ok := obj.(*v1.Codebase)
	if !ok {
		r.log.Info("the wrong object given, skipping validation")

		return nil
	}

	if createdCodebase.Spec.GitUrlPath == nil {
		r.log.Info("git url path is empty, skipping validation")

		return nil
	}

	codeBases := &v1.CodebaseList{}
	if err := r.client.List(ctx, codeBases, client.InNamespace(req.Namespace), client.Limit(listLimit)); err != nil {
		return fmt.Errorf("failed to list codebases: %w", err)
	}

	for i := range codeBases.Items {
		if codeBases.Items[i].Spec.GitUrlPath != nil && *codeBases.Items[i].Spec.GitUrlPath == *createdCodebase.Spec.GitUrlPath {
			return fmt.Errorf(
				"codebase %s with GitUrlPath %s already exists",
				codeBases.Items[i].Name,
				*createdCodebase.Spec.GitUrlPath,
			)
		}
	}

	return nil
}

// ValidateUpdate is a webhook for validating the updating of the Codebase CR.
func (r *CodebaseValidationWebhook) ValidateUpdate(ctx context.Context, _, _ runtime.Object) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("expected admission.Request in ctx: %w", err)
	}

	r.log.Info("validate update", "name", req.Name)

	return nil
}

// ValidateDelete is a webhook for validating the deleting of the Codebase CR.
// It is skipped for now. Add kubebuilder:webhook:verbs=delete to enable it.
func (r *CodebaseValidationWebhook) ValidateDelete(ctx context.Context, _ runtime.Object) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("expected admission.Request in ctx: %w", err)
	}

	r.log.Info("validate delete", "name", req.Name)

	return nil
}
