package stalecheck

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebasebranch"
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

func (a *CleanupAction) Apply(ctx context.Context, branch *codebaseApi.CodebaseBranch, verdict Verdict) error {
	if verdict.ExistsInGit {
		return a.mark.Apply(ctx, branch, verdict)
	}

	usage, err := codebasebranch.FindBranchUsage(ctx, a.client, branch)
	if err != nil {
		return fmt.Errorf("failed to check CodebaseBranch %s usage: %w", branch.Name, err)
	}

	if usage != "" {
		verdict.RetainedBy = usage

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
