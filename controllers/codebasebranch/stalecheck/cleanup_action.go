package stalecheck

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// CleanupAction implements the "auto" cleanup strategy: stale branches that do not
// participate in any deployment are deleted (the owned CodebaseImageStream is
// garbage-collected via its owner reference); branches referenced by a CDPipeline
// or Stage fall back to being marked, with the retaining resource recorded in the
// Stale condition message.
type CleanupAction struct {
	client   client.Client
	recorder record.EventRecorder
	mark     *MarkAction
}

func NewCleanupAction(k8sClient client.Client, recorder record.EventRecorder, mark *MarkAction) *CleanupAction {
	return &CleanupAction{client: k8sClient, recorder: recorder, mark: mark}
}

// Apply deletes the branch only when it is stale and unretained. Which deployment
// resources retain it is decided by the Checker, which resolves RetainedBy from the
// snapshot shared by every branch of the sweep.
func (a *CleanupAction) Apply(ctx context.Context, branch *codebaseApi.CodebaseBranch, verdict Verdict) error {
	if verdict.ExistsInGit || verdict.RetainedBy != "" {
		return a.mark.Apply(ctx, branch, verdict)
	}

	if a.recorder != nil {
		a.recorder.Eventf(branch, corev1.EventTypeNormal, EventReasonStaleBranchDeleted,
			"Branch %s was not found in the git repository and is not used by any deployment, deleting", branch.Spec.BranchName)
	}

	if err := a.client.Delete(ctx, branch); err != nil {
		return fmt.Errorf("failed to delete stale CodebaseBranch %s: %w", branch.Name, err)
	}

	ctrl.LoggerFrom(ctx).Info("Deleted stale codebase branch",
		"codebasebranch", branch.Name, "branch", branch.Spec.BranchName)

	return nil
}
