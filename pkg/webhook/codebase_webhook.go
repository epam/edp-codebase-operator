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

	v1 "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const listLimit = 1000

//+kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-codebase,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=codebases,verbs=create;update;delete,versions=v1,name=vcodebase.kb.io,admissionReviewVersions=v1

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
func (r *CodebaseValidationWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Errorf("expected admission.Request in ctx: %w", err).Error())
	}

	r.log.Info("validate create", "name", req.Name)

	createdCodebase, ok := obj.(*v1.Codebase)
	if !ok {
		r.log.Info("the wrong object given, skipping validation")

		return nil, nil
	}

	if err = validateCodBaseName(createdCodebase.Name); err != nil {
		return nil, err
	}

	if err = IsCodebaseValid(createdCodebase); err != nil {
		return nil, fmt.Errorf("codebase %s is invalid: %w", createdCodebase.Name, err)
	}

	gitUrlPathToValidate := util.TrimGitFromURL(createdCodebase.Spec.GitUrlPath)
	if gitUrlPathToValidate == "" {
		return nil, fmt.Errorf("gitUrlPath %s is invalid", createdCodebase.Spec.GitUrlPath)
	}

	codeBases := &v1.CodebaseList{}
	if err := r.client.List(ctx, codeBases, client.InNamespace(req.Namespace), client.Limit(listLimit)); err != nil {
		return nil, fmt.Errorf("failed to list codebases: %w", err)
	}

	for i := range codeBases.Items {
		gitUrlPath := codeBases.Items[i].Spec.GitUrlPath

		if gitUrlPath == gitUrlPathToValidate {
			return nil, fmt.Errorf(
				"codebase %s with GitUrlPath %s already exists",
				codeBases.Items[i].Name,
				gitUrlPath,
			)
		}
	}

	return nil, nil
}

// ValidateUpdate is a webhook for validating the updating of the Codebase CR.
func (r *CodebaseValidationWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}

	r.log.Info("validate update", "name", req.Name)

	updatedCodebase, ok := newObj.(*v1.Codebase)
	if !ok {
		r.log.Info("the wrong object given, skipping validation")

		return nil, nil
	}

	if err = IsCodebaseValid(updatedCodebase); err != nil {
		return nil, fmt.Errorf("codebase %s is invalid: %w", updatedCodebase.Name, err)
	}

	if err = checkResourceProtectionFromModificationOnUpdate(oldObj, newObj); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete is a webhook for validating the deleting of the Codebase CR.
func (r *CodebaseValidationWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}

	r.log.Info("validate delete", "name", req.Name)

	if err = checkResourceProtectionFromModificationOnDelete(obj); err != nil {
		return nil, err
	}

	return nil, nil
}
