package chain

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
)

type DeleteJiraIssueMetadataCr struct {
	next handler.JiraIssueMetadataHandler
	c    client.Client
}

func (h DeleteJiraIssueMetadataCr) ServeRequest(metadata *codebaseApi.JiraIssueMetadata) error {
	if metadata.Status.Error != nil {
		return errors.New(metadata.Status.PrintErrors())
	}

	logv := log.WithValues("name", metadata.Name)
	logv.V(2).Info("start deleting Jira issue metadata cr.")

	if err := h.c.Delete(context.TODO(), metadata); err != nil {
		return fmt.Errorf("failed to remove fix version cr %v: %w", metadata.Name, err)
	}

	logv.Info("Jira issue metadata cr has been deleted.")

	return nextServeOrNil(h.next, metadata)
}
