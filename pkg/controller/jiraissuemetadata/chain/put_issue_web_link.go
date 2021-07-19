package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
)

type PutIssueWebLink struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

func (h PutIssueWebLink) ServeRequest(metadata *v1alpha1.JiraIssueMetadata) error {
	log.Info("start creating web link in issues.")
	requestPayload, err := util.GetFieldsMap(metadata.Spec.Payload, nil)
	if err != nil {
		return errors.Wrap(err, "couldn't get map with Jira field values")
	}

	for _, linkInfo := range requestPayload["issuesLinks"].([]interface{}) {
		info := linkInfo.(map[string]interface{})
		if err := h.client.CreateIssueLink(info["ticket"].(string), info["title"].(string), info["url"].(string)); err != nil {
			return errors.Wrapf(err, "an error has occurred during creating remote link."+
				" ticket - %v, title - %v, url - %v", info["ticket"].(string), info["title"].(string), info["url"].(string))
		}
	}
	log.Info("end creating web link in issues.")
	return nextServeOrNil(h.next, metadata)
}
