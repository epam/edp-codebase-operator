package chain

import (
	"fmt"
	gojira "github.com/andygrunwald/go-jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/trivago/tgo/tcontainer"
	"strconv"
)

type PutTagValue struct {
	next   handler.JiraIssueMetadataHandler
	client jira.Client
}

const (
	issuesLinksKey     = "issuesLinks"
	jiraLabelFieldName = "labels"
)

func (h PutTagValue) ServeRequest(metadata *v1alpha1.JiraIssueMetadata) error {
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

	if err := h.tryToCreateFieldValues(requestPayload, metadata.Spec.Tickets[0], createApiMap); err != nil {
		return errors.Wrap(err, "an error has occurred during creating field values in Jira project")
	}

	log.Info("end creating field values in Jira project.")
	return nextServeOrNil(h.next, metadata)
}

func (h PutTagValue) tryToCreateFieldValues(requestPayload map[string]interface{}, ticket string,
	createApiMap map[string]func(projectId int, versionName string) error) error {
	projectInfo, err := h.client.GetProjectInfo(ticket)
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

	metaInfo := findIssueMetaInfo(issueTypes, *issueType)
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

func findProject(projects []*gojira.MetaProject, id string) *gojira.MetaProject {
	for _, p := range projects {
		if p.Id == id {
			return p
		}
	}
	return nil
}

func findIssueMetaInfo(types []*gojira.MetaIssueType, issueType string) tcontainer.MarshalMap {
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
