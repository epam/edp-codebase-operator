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
