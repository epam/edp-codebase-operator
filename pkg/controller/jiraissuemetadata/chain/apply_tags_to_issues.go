package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
)

type ApplyTagsToIssues struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

func (h ApplyTagsToIssues) ServeRequest(metadata *v1alpha1.JiraIssueMetadata) error {
	log.Info("start applying tags to issues.")
	requestPayload, err := util.GetFieldsMap(metadata.Spec.Payload, []string{issuesLinksKey})
	if err != nil {
		return errors.Wrap(err, "couldn't get map with Jira field values")
	}

	body := createRequestBody(requestPayload)
	for _, ticket := range metadata.Spec.Tickets {
		if err := h.client.ApplyTagsToIssue(ticket, body); err != nil {
			return errors.Wrapf(err, "couldn't apply tags to issue %v", ticket)
		}
	}
	log.Info("end applying tags to issues.")
	return nextServeOrNil(h.next, metadata)
}

func createRequestBody(requestPayload map[string]interface{}) map[string]interface{} {
	params := map[string]interface{}{
		"update": map[string]interface{}{},
	}

	for k, v := range requestPayload {
		if k == "labels" {
			params["update"].(map[string]interface{})[k] = []map[string]interface{}{
				{
					"add": v.(string),
				},
			}
		} else {
			params["update"].(map[string]interface{})[k] = []map[string]interface{}{
				{"add": struct {
					Name string `json:"name"`
				}{
					v.(string),
				},
				},
			}
		}
	}
	return params
}
