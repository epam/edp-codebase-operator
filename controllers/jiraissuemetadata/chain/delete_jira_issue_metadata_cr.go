package chain

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
)

type DeleteJiraIssueMetadataCr struct {
	next handler.JiraIssueMetadataHandler
	c    client.Client
}

func (h DeleteJiraIssueMetadataCr) ServeRequest(metadata *codebaseApi.JiraIssueMetadata) error {
	logv := log.WithValues("name", metadata.Name)
	logv.V(2).Info("start deleting Jira issue metadata cr.")

	if err := h.c.Delete(context.TODO(), metadata); err != nil {
		return errors.Wrapf(err, "couldn't remove fix version cr %v", metadata.Name)
	}

	logv.Info("Jira issue metadata cr has been deleted.")

	return nextServeOrNil(h.next, metadata)
}