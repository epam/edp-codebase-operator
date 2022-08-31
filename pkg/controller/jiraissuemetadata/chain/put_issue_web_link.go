package chain

import (
	"fmt"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type PutIssueWebLink struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

func (h PutIssueWebLink) ServeRequest(metadata *codebaseApi.JiraIssueMetadata) error {
	log.Info("start creating web link in issues.")
	requestPayload, err := util.GetFieldsMap(metadata.Spec.Payload, nil)
	if err != nil {
		return errors.Wrap(err, "couldn't get map with Jira field values")
	}
	if _, ok := requestPayload["issuesLinks"]; !ok {
		return errors.New("issuesLinks is a mandatory field in payload")
	}

	for _, linkInfo := range requestPayload["issuesLinks"].([]interface{}) {
		info := linkInfo.(map[string]interface{})
		if err := h.client.CreateIssueLink(info["ticket"].(string), info["title"].(string), info["url"].(string)); err != nil {
			metadata.Status.Error = multierror.Append(metadata.Status.Error,
				fmt.Errorf("an error has occurred during creating remote link."+
					" ticket - %s, title - %s, url - %s, err: %w", info["ticket"].(string), info["title"].(string),
					info["url"].(string), err))
		}
	}

	log.Info("end creating web link in issues.")
	return nextServeOrNil(h.next, metadata)
}
