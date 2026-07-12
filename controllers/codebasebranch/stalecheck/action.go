package stalecheck

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	EventReasonBranchStale         = "BranchStale"
	EventReasonBranchStaleResolved = "BranchStaleResolved"
	EventReasonStaleBranchDeleted  = "StaleBranchDeleted"
	EventReasonStaleBranchRetained = "StaleBranchRetained"
)

// Verdict is the result of checking a single CodebaseBranch against the git repository.
type Verdict struct {
	ExistsInGit bool

	// RetainedBy describes the deployment resource that prevents automatic cleanup
	// of a missing branch (empty when the branch exists or is not retained).
	RetainedBy string
}

// StaleBranchAction applies a cleanup strategy decision to a CodebaseBranch based on a Verdict.
type StaleBranchAction interface {
	Apply(ctx context.Context, branch *codebaseApi.CodebaseBranch, verdict Verdict) error
}

// MarkAction marks/unmarks a CodebaseBranch as stale via the Stale status condition
// (source of truth) and the mirrored stale label (for label-selector filtering).
type MarkAction struct {
	client   client.Client
	recorder record.EventRecorder
}

func NewMarkAction(k8sClient client.Client, recorder record.EventRecorder) *MarkAction {
	return &MarkAction{client: k8sClient, recorder: recorder}
}

func (a *MarkAction) Apply(ctx context.Context, branch *codebaseApi.CodebaseBranch, verdict Verdict) error {
	wasStale := meta.IsStatusConditionTrue(branch.Status.Conditions, codebaseApi.ConditionStale)

	if err := a.updateStatusCondition(ctx, branch, verdict); err != nil {
		return err
	}

	if err := a.updateStaleLabel(ctx, branch, verdict); err != nil {
		return err
	}

	a.emitTransitionEvent(branch, wasStale, verdict)

	return nil
}

func (a *MarkAction) updateStatusCondition(
	ctx context.Context,
	branch *codebaseApi.CodebaseBranch,
	verdict Verdict,
) error {
	// Messages are deliberately static so that SetStatusCondition reports a change
	// (and triggers a status write) only on real transitions, not on every periodic check.
	condition := metav1.Condition{
		Type:               codebaseApi.ConditionStale,
		Status:             metav1.ConditionFalse,
		Reason:             codebaseApi.ReasonBranchFoundInGit,
		Message:            "Branch exists in the git repository",
		ObservedGeneration: branch.Generation,
	}

	if !verdict.ExistsInGit {
		condition.Status = metav1.ConditionTrue
		condition.Reason = codebaseApi.ReasonBranchNotFoundInGit
		condition.Message = "Branch was not found in the git repository"

		if verdict.RetainedBy != "" {
			condition.Message = fmt.Sprintf(
				"Branch was not found in the git repository; retained because it is used by %s", verdict.RetainedBy)
		}
	}

	original := branch.DeepCopy()
	if !meta.SetStatusCondition(&branch.Status.Conditions, condition) {
		return nil
	}

	if err := a.client.Status().Patch(ctx, branch, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch Stale condition on CodebaseBranch %s: %w", branch.Name, err)
	}

	return nil
}

func (a *MarkAction) updateStaleLabel(ctx context.Context, branch *codebaseApi.CodebaseBranch, verdict Verdict) error {
	value, labeled := branch.Labels[codebaseApi.StaleLabel]

	upToDate := (verdict.ExistsInGit && !labeled) || (!verdict.ExistsInGit && value == "true")
	if upToDate {
		return nil
	}

	original := branch.DeepCopy()

	if verdict.ExistsInGit {
		delete(branch.Labels, codebaseApi.StaleLabel)
	} else {
		if branch.Labels == nil {
			branch.Labels = make(map[string]string, 1)
		}

		branch.Labels[codebaseApi.StaleLabel] = "true"
	}

	if err := a.client.Patch(ctx, branch, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch stale label on CodebaseBranch %s: %w", branch.Name, err)
	}

	return nil
}

func (a *MarkAction) emitTransitionEvent(branch *codebaseApi.CodebaseBranch, wasStale bool, verdict Verdict) {
	if a.recorder == nil {
		return
	}

	if !wasStale && !verdict.ExistsInGit {
		if verdict.RetainedBy != "" {
			a.recorder.Eventf(branch, corev1.EventTypeWarning, EventReasonStaleBranchRetained,
				"Branch %s was not found in the git repository; it is marked as stale but retained because it is used by %s",
				branch.Spec.BranchName, verdict.RetainedBy)

			return
		}

		a.recorder.Eventf(branch, corev1.EventTypeWarning, EventReasonBranchStale,
			"Branch %s was not found in the git repository and is marked as stale", branch.Spec.BranchName)
	}

	if wasStale && verdict.ExistsInGit {
		a.recorder.Eventf(branch, corev1.EventTypeNormal, EventReasonBranchStaleResolved,
			"Branch %s exists in the git repository again, stale mark removed", branch.Spec.BranchName)
	}
}
