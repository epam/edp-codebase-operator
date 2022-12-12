package chain

import (
	"github.com/pkg/errors"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type ApplyTagsToIssues struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

func (h ApplyTagsToIssues) ServeRequest(metadata *codebaseApi.JiraIssueMetadata) error {
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
	updateField := map[string]interface{}{}

	for k, v := range requestPayload {
		strVal, ok := v.(string)
		if !ok {
			continue
		}

		if k == "labels" {
			updateField[k] = []map[string]interface{}{
				{
					"add": strVal,
				},
			}

			continue
		}

		updateField[k] = []map[string]interface{}{
			{"add": struct {
				Name string `json:"name"`
			}{
				strVal,
			},
			},
		}
	}

	return map[string]interface{}{
		"update": updateField,
	}
}
