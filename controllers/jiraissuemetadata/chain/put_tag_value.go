package chain

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	goJira "github.com/andygrunwald/go-jira"
	"golang.org/x/exp/slices"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutTagValue struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

const (
	issuesLinksKey     = "issuesLinks"
	jiraLabelFieldName = "labels"
)

func (h PutTagValue) ServeRequest(ctx context.Context, metadata *codebaseApi.JiraIssueMetadata) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creating field values in Jira project")

	requestPayload, err := util.GetFieldsMap(metadata.Spec.Payload, []string{issuesLinksKey, jiraLabelFieldName})
	if err != nil {
		return fmt.Errorf("failed to get map with Jira field values: %w", err)
	}

	createApiMap := map[string]func(ctx context.Context, projectId int, versionName string) error{
		"fixVersions": h.client.CreateFixVersionValue,
		"components":  h.client.CreateComponentValue,
	}

	if len(metadata.Spec.Tickets) == 0 {
		return errors.New("JiraIssueMetadata is invalid. Tickets field can't be empty")
	}

	if err := h.tryToCreateFieldValues(ctx, requestPayload, metadata.Spec.Tickets, createApiMap); err != nil {
		return fmt.Errorf("failed to create field values in Jira project: %w", err)
	}

	log.Info("end creating field values in Jira project.")

	return nextServeOrNil(ctx, h.next, metadata)
}

func (h PutTagValue) tryToCreateFieldValues(
	ctx context.Context,
	requestPayload map[string]interface{},
	tickets []string,
	createApiMap map[string]func(ctx context.Context, projectId int, versionName string) error,
) error {
	log := ctrl.LoggerFrom(ctx)

	projectInfo, ticket, err := h.getProjectInfo(tickets)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	issue, err := h.client.GetIssue(ctx, ticket)
	if err != nil {
		return fmt.Errorf("failed to get issue for %v ticket: %w", ticket, err)
	}

	issueTypeMeta, err := h.client.GetIssueTypeMeta(ctx, projectInfo.ID, issue.Fields.Type.ID)
	if err != nil {
		return fmt.Errorf("failed to get issue type meta: %w", err)
	}

	for k, v := range requestPayload {
		metaInfo, ok := issueTypeMeta[k]
		if !ok {
			log.Info(fmt.Sprintf("Issue type %s doesn't contain field %s", issue.Fields.Type.Name, k))
			continue
		}

		val, ok := v.(string)
		if !ok {
			return fmt.Errorf("wrong type of payload value, '%v' is not string", v)
		}

		if slices.ContainsFunc(metaInfo.AllowedValues, func(value jira.IssueTypeMetaAllowedValue) bool {
			return value.Name == val
		}) {
			log.Info(fmt.Sprintf("Issue type %s field %s already contains value %s", issue.Fields.Type.Name, k, val))
			continue
		}

		id, err := strconv.Atoi(projectInfo.ID)
		if err != nil {
			return fmt.Errorf("failed to parse to int project ID: %w", err)
		}

		if err = createApiMap[k](ctx, id, val); err != nil {
			return fmt.Errorf("failed to create value %v for %v field: %w", val, k, err)
		}
	}

	return nil
}

func (h PutTagValue) getProjectInfo(tickets []string) (*goJira.Project, string, error) {
	for _, ticket := range tickets {
		projectInfo, err := h.client.GetProjectInfo(ticket)
		if err != nil {
			if errors.Is(err, jira.ErrNotFound) {
				continue
			}

			return nil, "", fmt.Errorf("failed to get project info: %w", err)
		}

		return projectInfo, ticket, nil
	}

	return nil, "", errors.New("jira issue not found")
}
