package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
)

type DeleteJiraIssueMetadataCr struct {
	next handler.JiraIssueMetadataHandler
	c    client.Client
}

func (h DeleteJiraIssueMetadataCr) ServeRequest(ctx context.Context, metadata *codebaseApi.JiraIssueMetadata) error {
	log := ctrl.LoggerFrom(ctx)

	if metadata.Status.ErrorStrings != nil {
		return errors.New(strings.Join(metadata.Status.ErrorStrings, "\n"))
	}

	log.Info("Start deleting Jira issue metadata cr")

	if err := h.c.Delete(ctx, metadata); err != nil {
		return fmt.Errorf("failed to remove fix version cr %v: %w", metadata.Name, err)
	}

	log.Info("Jira issue metadata cr has been deleted")

	return nextServeOrNil(ctx, h.next, metadata)
}
