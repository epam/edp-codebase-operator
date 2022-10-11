package chain

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("jira_issue_metadata_handler")

const issuesLinks = "issuesLinks"

func CreateChain(metadataPayload string, jiraClient jira.Client, c client.Client) (handler.JiraIssueMetadataHandler, error) {
	payload, err := util.GetFieldsMap(metadataPayload, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fields map from payload: %w", err)
	}

	if len(payload) == 1 && payload[issuesLinks] != nil {
		return createWithoutApplyingTagsChain(jiraClient, c), nil
	}

	return createDefChain(jiraClient, c), nil
}

func createDefChain(jiraClient jira.Client, c client.Client) handler.JiraIssueMetadataHandler {
	return PutTagValue{
		next: ApplyTagsToIssues{
			next: PutIssueWebLink{
				next: DeleteJiraIssueMetadataCr{
					c: c,
				},
				client: jiraClient,
			},
			client: jiraClient,
		},
		client: jiraClient,
	}
}

func createWithoutApplyingTagsChain(jiraClient jira.Client, c client.Client) handler.JiraIssueMetadataHandler {
	return PutIssueWebLink{
		next: DeleteJiraIssueMetadataCr{
			c: c,
		},
		client: jiraClient,
	}
}

func nextServeOrNil(next handler.JiraIssueMetadataHandler, metadata *codebaseApi.JiraIssueMetadata) error {
	if next == nil {
		log.Info("handling of JiraIssueMetadata has been finished", "name", metadata.Name)

		return nil
	}

	err := next.ServeRequest(metadata)
	if err != nil {
		return fmt.Errorf("failed to process next handler: %w", err)
	}

	return nil
}
