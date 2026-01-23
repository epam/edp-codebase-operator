package webhook

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
)

// +kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-codebasebranch,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=codebasebranches,verbs=create;update;delete,versions=v1,name=codebasebranch.epam.com,admissionReviewVersions=v1

// CodebaseBranchValidationWebhook is a webhook for validating CodebaseBranch CRD.
type CodebaseBranchValidationWebhook struct {
	client client.Client
	log    logr.Logger
}

// NewCodebaseBranchValidationWebhook creates a new webhook for validating CodebaseBranch CR.
func NewCodebaseBranchValidationWebhook(k8sClient client.Client, log logr.Logger) *CodebaseBranchValidationWebhook {
	return &CodebaseBranchValidationWebhook{client: k8sClient, log: log.WithName("codebasebranch-webhook")}
}

// SetupWebhookWithManager sets up the webhook with the manager for CodebaseBranch CR.
func (r *CodebaseBranchValidationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&v1.CodebaseBranch{}).
		WithValidator(r).
		Complete()
	if err != nil {
		return fmt.Errorf("failed to build CodebaseBranch validation webhook: %w", err)
	}

	return nil
}

var _ webhook.CustomValidator = &CodebaseBranchValidationWebhook{}

// ValidateCreate is a webhook for validating the creation of the CodebaseBranch CR.
func (r *CodebaseBranchValidationWebhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (warnings admission.Warnings, err error) {
	createdCodebaseBranch, ok := obj.(*v1.CodebaseBranch)
	if !ok {
		r.log.Info("The wrong object given, skipping validation")

		return nil, nil
	}

	list := &v1.CodebaseBranchList{}
	if err = r.client.List(
		ctx,
		list,
		client.InNamespace(createdCodebaseBranch.Namespace),
		client.MatchingLabels{
			v1.CodebaseLabel:   createdCodebaseBranch.Spec.CodebaseName,
			v1.BranchHashLabel: codebasebranch.MakeGitBranchHash(createdCodebaseBranch.Spec.BranchName),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to list CodebaseBranch CRs: %w", err)
	}

	if len(list.Items) > 0 {
		return nil, fmt.Errorf("CodebaseBranch CR with the same codebase name and branch name already exists")
	}

	return nil, nil
}

// ValidateUpdate is a webhook for validating the updating of the CodebaseBranch CR.
func (r *CodebaseBranchValidationWebhook) ValidateUpdate(
	ctx context.Context,
	oldObj, newObj runtime.Object,
) (warnings admission.Warnings, err error) {
	if err = checkResourceProtectionFromModificationOnUpdate(oldObj, newObj); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete is a webhook for validating the deleting of the CodebaseBranch CR.
func (r *CodebaseBranchValidationWebhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) (warnings admission.Warnings, err error) {
	if err = checkResourceProtectionFromModificationOnDelete(obj); err != nil {
		return nil, err
	}

	return nil, nil
}
