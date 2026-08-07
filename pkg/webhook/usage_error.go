package webhook

import (
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/epam/edp-codebase-operator/v2/pkg/deploymentusage"
)

const (
	codebaseKind       = "Codebase"
	codebaseBranchKind = "CodebaseBranch"
)

// newBlockedByUsageError builds a StatusError that denies deletion of a
// resource still referenced by deployment resources.
//
// It intentionally hand-builds the status rather than using
// apierrors.NewForbidden/NewInvalid: NewForbidden does not carry
// Details.Causes, and NewInvalid returns HTTP 422/"Invalid" semantics which
// is wrong here (the object being deleted is not invalid, it is blocked by
// external references) and would change the HTTP code clients rely on.
// controller-runtime's admission handler unwraps *apierrors.StatusError via
// errors.As and forwards its full metav1.Status, so Details.Causes reaches
// the client intact.
func newBlockedByUsageError(
	gr schema.GroupResource,
	kind string,
	name string,
	refs []deploymentusage.Reference,
) *apierrors.StatusError {
	causes := make([]metav1.StatusCause, 0, len(refs))

	for _, ref := range refs {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeForbidden,
			Field:   ref.Field,
			Message: ref.String(),
		})
	}

	message := fmt.Sprintf(
		"%s %s cannot be deleted because it is used by %s; remove it from the deployment first",
		kind, name, deploymentusage.Join(refs),
	)

	return &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    http.StatusForbidden,
			Reason:  metav1.StatusReasonForbidden,
			Message: message,
			Details: &metav1.StatusDetails{
				Group:  gr.Group,
				Kind:   gr.Resource,
				Name:   name,
				Causes: causes,
			},
		},
	}
}
