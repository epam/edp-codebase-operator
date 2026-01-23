package webhook

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	protectedLabel  = "app.edp.epam.com/edit-protection"
	deleteOperation = "delete"
	updateOperation = "update"
)

// +kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-gitserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=gitservers,verbs=update;delete,versions=v1,name=gitserver.epam.com,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-codebaseimagestream,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=codebaseimagestreams,verbs=update;delete,versions=v1,name=codebaseimagestream.epam.com,admissionReviewVersions=v1

// ProtectedLabelValidationWebhook is a webhook for validating ProtectedLabel CRD.
type ProtectedLabelValidationWebhook struct {
}

// SetupWebhookWithManager sets up the webhook with the manager for given resources.
func (r *ProtectedLabelValidationWebhook) SetupWebhookWithManager(mgr ctrl.Manager, objects ...runtime.Object) error {
	for _, obj := range objects {
		err := ctrl.NewWebhookManagedBy(mgr).
			For(obj).
			WithValidator(r).
			Complete()
		if err != nil {
			return fmt.Errorf("failed to build %s validation webhook: %w", obj.GetObjectKind().GroupVersionKind(), err)
		}
	}

	return nil
}

// ValidateCreate is a webhook for validating the creation of the ProtectedLabel CR.
func (*ProtectedLabelValidationWebhook) ValidateCreate(
	_ context.Context,
	_ runtime.Object,
) (warnings admission.Warnings, err error) {
	return nil, nil
}

// ValidateUpdate is a webhook for validating the updating of the ProtectedLabel CR.
func (*ProtectedLabelValidationWebhook) ValidateUpdate(
	_ context.Context,
	oldObj, newObj runtime.Object,
) (warnings admission.Warnings, err error) {
	if err = checkResourceProtectionFromModificationOnUpdate(oldObj, newObj); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete is a webhook for validating the deleting of the ProtectedLabel CR.
func (*ProtectedLabelValidationWebhook) ValidateDelete(
	_ context.Context,
	obj runtime.Object,
) (warnings admission.Warnings, err error) {
	if err = checkResourceProtectionFromModificationOnDelete(obj); err != nil {
		return nil, err
	}

	return nil, nil
}

func hasProtectedLabel(obj runtime.Object, operation string) bool {
	o, ok := obj.(metaV1.Object)
	if !ok {
		return false
	}

	return o.GetLabels()[protectedLabel] != "" &&
		slices.Contains(strings.Split(o.GetLabels()[protectedLabel], "-"), operation)
}

func isSpecUpdated(oldObj, newObj runtime.Object) bool {
	switch old := oldObj.(type) {
	case *codebaseApi.Codebase:
		if newCodebase, ok := newObj.(*codebaseApi.Codebase); ok {
			return !equality.Semantic.DeepEqual(old.Spec, newCodebase.Spec)
		}
	case *codebaseApi.CodebaseBranch:
		if newBranch, ok := newObj.(*codebaseApi.CodebaseBranch); ok {
			return !equality.Semantic.DeepEqual(old.Spec, newBranch.Spec)
		}
	case *codebaseApi.CodebaseImageStream:
		if newImageStream, ok := newObj.(*codebaseApi.CodebaseImageStream); ok {
			return !equality.Semantic.DeepEqual(old.Spec, newImageStream.Spec)
		}
	case *codebaseApi.GitServer:
		if newGitServer, ok := newObj.(*codebaseApi.GitServer); ok {
			return !equality.Semantic.DeepEqual(old.Spec, newGitServer.Spec)
		}
	}

	return false
}

func checkResourceProtectionFromModificationOnDelete(obj runtime.Object) error {
	if hasProtectedLabel(obj, deleteOperation) {
		return errors.New("resource contains label that protects it from deletion")
	}

	return nil
}

func checkResourceProtectionFromModificationOnUpdate(oldObj, newObj runtime.Object) error {
	if hasProtectedLabel(newObj, updateOperation) && isSpecUpdated(oldObj, newObj) {
		return errors.New("resource contains label that protects it from modification")
	}

	return nil
}
