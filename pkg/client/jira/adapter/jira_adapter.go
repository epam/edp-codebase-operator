package adapter

import (
	"github.com/andygrunwald/go-jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("gojira_adapter")

type GoJiraAdapter struct {
	client jira.Client
}

func (a GoJiraAdapter) Connected() (bool, error) {
	log.V(2).Info("start Connected method")
	user, _, err := a.client.User.GetSelf()
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

func (a GoJiraAdapter) FixVersionExists(projectId, version string) (bool, error) {
	logv := log.WithValues("project", projectId, "fix version", version)
	logv.V(2).Info("start GetFixVersion method.", "project id", projectId, "name", version)
	project, _, err := a.client.Project.Get(projectId)
	if err != nil {
		return false, err
	}
	for _, v := range project.Versions {
		if v.Name == version {
			logv.V(2).Info("fix version already exists in project.")
			return true, nil
		}
	}
	logv.V(2).Info("fix version doesnt exist in project.")
	return false, nil
}

func (a GoJiraAdapter) GetProjectId(issue string) (*string, error) {
	logv := log.WithValues("issue", issue)
	logv.V(2).Info("start GetProjectId method.")
	issueResp, _, err := a.client.Issue.Get(issue, nil)
	if err != nil {
		return nil, err
	}
	logv.V(2).Info("project id has been fetched.", "id", issueResp.Fields.Project.ID)
	return util.GetStringP(issueResp.Fields.Project.ID), nil
}

func (a GoJiraAdapter) CreateFixVersion(projectId int, versionName string) error {
	logv := log.WithValues("project id", projectId, "version name", versionName)
	logv.V(2).Info("start CreateFixVersion method.")
	_, _, err := a.client.Version.Create(&jira.Version{
		Name:      versionName,
		ProjectID: projectId,
	})
	if err != nil {
		return err
	}
	logv.Info("fix version has been created.")
	return nil
}

func (a GoJiraAdapter) SetFixVersionToIssue(issue, fixVersion string) error {
	logv := log.WithValues("issue", issue, "fix version", fixVersion)
	logv.V(2).Info("start SetFixVersionToIssue method.")
	params := map[string]interface{}{
		"update": map[string]interface{}{
			"fixVersions": []map[string]interface{}{
				{"add": jira.FixVersion{
					Name: fixVersion,
				}},
			},
		},
	}
	if _, err := a.client.Issue.UpdateIssue(issue, params); err != nil {
		return err
	}
	logv.Info("fix version has been saved.")
	return nil
}
