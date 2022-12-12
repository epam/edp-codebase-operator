package chain

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
)

type link struct {
	Ticket string `json:"ticket"`
	Title  string `json:"title"`
	Url    string `json:"url"`
}

type jiraPayload struct {
	IssuesLinks []link `json:"issuesLinks,omitempty"`
}

type PutIssueWebLink struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

func (h PutIssueWebLink) ServeRequest(metadata *codebaseApi.JiraIssueMetadata) error {
	log.Info("start creating web link in issues.")

	payload := jiraPayload{}

	err := json.Unmarshal([]byte(metadata.Spec.Payload), &payload)
	if err != nil {
		return fmt.Errorf("invalid spec payload json: %w", err)
	}

	if payload.IssuesLinks == nil {
		return errors.New("issuesLinks is a mandatory field in payload")
	}

	for _, linkInfo := range payload.IssuesLinks {
		if err = h.client.CreateIssueLink(linkInfo.Ticket, linkInfo.Title, linkInfo.Url); err != nil {
			return errors.Wrapf(err, "an error has occurred during creating remote link."+
				" ticket - %v, title - %v, url - %v", linkInfo.Ticket, linkInfo.Title, linkInfo.Url)
		}
	}

	log.Info("end creating web link in issues.")

	return nextServeOrNil(h.next, metadata)
}
