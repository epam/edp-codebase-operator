package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jira_issue_metadata_handler")

const issuesLinks = "issuesLinks"

func CreateChain(metadataPayload string, jiraClient *jira.Client, client client.Client) (handler.JiraIssueMetadataHandler, error) {
	payload, err := util.GetFieldsMap(metadataPayload, nil)
	if err != nil {
		return nil, err
	}

	if len(payload) == 1 && payload[issuesLinks] != nil {
		return createWithoutApplyingTagsChain(jiraClient, client), nil
	}

	return createDefChain(jiraClient, client), nil
}

func createDefChain(jiraClient *jira.Client, client client.Client) handler.JiraIssueMetadataHandler {
	return PutTagValue{
		next: ApplyTagsToIssues{
			next: PutIssueWebLink{
				next: DeleteJiraIssueMetadataCr{
					c: client,
				},
				client: *jiraClient,
			},
			client: *jiraClient,
		},
		client: *jiraClient,
	}
}

func createWithoutApplyingTagsChain(jiraClient *jira.Client, client client.Client) handler.JiraIssueMetadataHandler {
	return PutIssueWebLink{
		next: DeleteJiraIssueMetadataCr{
			c: client,
		},
		client: *jiraClient,
	}
}

func nextServeOrNil(next handler.JiraIssueMetadataHandler, metadata *v1alpha1.JiraIssueMetadata) error {
	if next != nil {
		return next.ServeRequest(metadata)
	}
	log.Info("handling of JiraIssueMetadata has been finished", "name", metadata.Name)
	return nil
}
