package chain

import (
	"context"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteJiraIssueMetadataCr struct {
	next handler.JiraIssueMetadataHandler
	c    client.Client
}

func (h DeleteJiraIssueMetadataCr) ServeRequest(metadata *v1alpha1.JiraIssueMetadata) error {
	logv := log.WithValues("name", metadata.Name)
	logv.V(2).Info("start deleting Jira issue metadata cr.")

	if err := h.c.Delete(context.TODO(), metadata); err != nil {
		return errors.Wrapf(err, "couldn't remove fix version cr %v.", metadata.Name)
	}

	logv.Info("Jira issue metadata cr has been deleted.")
	return nextServeOrNil(h.next, metadata)
}
