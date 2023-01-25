package chain

import (
	"fmt"
	"strconv"

	goJira "github.com/andygrunwald/go-jira"
	gojira "github.com/andygrunwald/go-jira"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/trivago/tgo/tcontainer"
)

type PutTagValue struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

const (
	issuesLinksKey     = "issuesLinks"
	jiraLabelFieldName = "labels"
)

func (h PutTagValue) ServeRequest(metadata *codebaseApi.JiraIssueMetadata) error {
	log.Info("start creating field values in Jira project.")
	requestPayload, err := util.GetFieldsMap(metadata.Spec.Payload, []string{issuesLinksKey, jiraLabelFieldName})
	if err != nil {
		return errors.Wrap(err, "couldn't get map with Jira field values")
	}

	createApiMap := map[string]func(projectId int, versionName string) error{
		"fixVersions": h.client.CreateFixVersionValue,
		"components":  h.client.CreateComponentValue,
	}

	if len(metadata.Spec.Tickets) == 0 {
		return errors.New("JiraIssueMetadata is invalid. Tickets field cann't be empty.")
	}

	if err := h.tryToCreateFieldValues(requestPayload, metadata.Spec.Tickets, createApiMap); err != nil {
		return errors.Wrap(err, "an error has occurred during creating field values in Jira project")
	}

	log.Info("end creating field values in Jira project.")
	return nextServeOrNil(h.next, metadata)
}

func (h PutTagValue) tryToCreateFieldValues(requestPayload map[string]interface{}, tickets []string,
	createApiMap map[string]func(projectId int, versionName string) error) error {

	projectInfo, ticket, err := h.getProjectInfo(tickets)
	if err != nil {
		return errors.Wrap(err, "couldn't get project info")
	}

	issueTypes, err := h.getIssueTypes(projectInfo.ID, projectInfo.Key)
	if err != nil {
		return errors.Wrap(err, "couldn't get list of issue types")
	}

	issueType, err := h.client.GetIssueType(ticket)
	if err != nil {
		return errors.Wrapf(err, "couldn't get issue type for %v ticket", ticket)
	}

	metaInfo := findIssueMetaInfo(issueTypes, issueType)
	for k, v := range requestPayload {
		fieldProperties, exists := metaInfo.Value(k)
		if !exists {
			log.Info("project doesnt contain field", "field", k)
			continue
		}

		allowedValues := fieldProperties.(map[string]interface{})["allowedValues"].([]interface{})
		val := v.(string)
		if !doesJiraContainValue(val, allowedValues) {
			log.Info("Jira doesn't contain value field. try to create.", "value", val)
			id, err := strconv.Atoi(projectInfo.ID)
			if err != nil {
				return err
			}
			if err := createApiMap[k](id, val); err != nil {
				return errors.Wrapf(err, "couldn't create value %v for %v field", val, k)
			}
		}
	}
	return nil
}

func (h PutTagValue) getProjectInfo(tickets []string) (*gojira.Project, string, error) {
	for _, ticket := range tickets {
		projectInfo, err := h.client.GetProjectInfo(ticket)
		if err != nil {
			if errors.Is(err, adapter.ErrNotFound) {
				continue
			}

			return nil, "", fmt.Errorf("unable to get project info: %w", err)
		}

		return projectInfo, ticket, nil
	}

	return nil, "", errors.New("jira issue not found")
}

func (h PutTagValue) getIssueTypes(projectId, projectKey string) ([]*gojira.MetaIssueType, error) {
	issueMetadata, err := h.client.GetIssueMetadata(projectKey)
	if err != nil {
		return nil, err
	}

	metaProject := findProject(issueMetadata.Projects, projectId)
	if metaProject == nil {
		return nil, fmt.Errorf("project with %v was not found", projectId)
	}
	return metaProject.IssueTypes, nil
}

func findProject(projects []*goJira.MetaProject, id string) *goJira.MetaProject {
	for _, p := range projects {
		if p.Id == id {
			return p
		}
	}
	return nil
}

func findIssueMetaInfo(types []*goJira.MetaIssueType, issueType string) tcontainer.MarshalMap {
	for _, t := range types {
		if t.Name == issueType {
			return t.Fields
		}
	}
	return nil
}

func doesJiraContainValue(value string, values []interface{}) bool {
	for _, v := range values {
		if value == v.(map[string]interface{})["name"] {
			return true
		}
	}
	return false
}
