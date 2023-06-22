package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type ApplyTagsToIssues struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

func (h ApplyTagsToIssues) ServeRequest(ctx context.Context, metadata *codebaseApi.JiraIssueMetadata) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start applying tags to issues")

	requestPayload, err := util.GetFieldsMap(metadata.Spec.Payload, []string{issuesLinksKey})
	if err != nil {
		return fmt.Errorf("failed to get map with Jira field values: %w", err)
	}

	body := createRequestBody(requestPayload)
	for _, ticket := range metadata.Spec.Tickets {
		if err := h.client.ApplyTagsToIssue(ticket, body); err != nil {
			metadata.Status.ErrorStrings = append(metadata.Status.ErrorStrings,
				fmt.Sprintf("failed to apply tags to issue %s, err: %v", ticket, err))
		}
	}

	log.Info("End applying tags to issues")

	return nextServeOrNil(ctx, h.next, metadata)
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
			{
				"add": struct {
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
