package chain

import (
	"context"
	"fmt"
	"strconv"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func setIntermediateSuccessFields(ctx context.Context, c client.Client, cb *codebaseApi.Codebase, action codebaseApi.ActionType) error {
	// Set WebHookRef from WebHookID for backward compatibility.
	webHookRef := cb.Status.WebHookRef
	if webHookRef == "" && cb.Status.WebHookID != 0 {
		webHookRef = strconv.Itoa(cb.Status.WebHookID)
	}

	cb.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Action:          action,
		Result:          codebaseApi.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    cb.Status.FailureCount,
		Git:             cb.Status.Git,
		WebHookID:       cb.Status.WebHookID,
		WebHookRef:      webHookRef,
		GitWebUrl:       cb.Status.GitWebUrl,
	}

	if err := c.Status().Update(ctx, cb); err != nil {
		return fmt.Errorf("failed to update status field of %q resource 'Codebase': %w", cb.Name, err)
	}

	return nil
}

// updateGitStatusWithPatch updates the codebase Git status using Patch instead of Update.
// If a conflict occurs, the function returns an error, causing the reconciliation
// to requeue automatically via the controller-runtime framework.
func updateGitStatusWithPatch(
	ctx context.Context,
	c client.Client,
	codebase *codebaseApi.Codebase,
	action codebaseApi.ActionType,
	gitStatus string,
) error {
	// Skip update if status already matches (idempotency check)
	if codebase.Status.Git == gitStatus {
		return nil
	}

	// Create patch based on current object state
	patch := client.MergeFrom(codebase.DeepCopy())

	// Modify the status field
	codebase.Status.Git = gitStatus

	// Apply patch to status subresource
	if err := c.Status().Patch(ctx, codebase, patch); err != nil {
		setFailedFields(codebase, action, err.Error())
		return fmt.Errorf("failed to patch git status to %s for codebase %s: %w",
			gitStatus, codebase.Name, err)
	}

	return nil
}
