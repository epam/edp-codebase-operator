package gitlab

import (
	"os"
	"testing"
)

type gitlab struct {
	url                       string
	user                      string
	pass                      string
	token                     string
	group                     string
	groupId                   string
	existingProjectInGroup    string
	nonExistingProjectInGroup string
	sshUrl                    string
	nonExistingGroup          string
}

func createGitlab() gitlab {
	return gitlab{
		url:                       os.Getenv("TEST_GITLAB_URL"),
		user:                      os.Getenv("TEST_GITLAB_USERNAME"),
		pass:                      os.Getenv("TEST_GITLAB_PASSWORD"),
		token:                     os.Getenv("TEST_GITLAB_TOKEN"),
		group:                     os.Getenv("TEST_GITLAB_GROUP"),
		groupId:                   os.Getenv("TEST_GITLAB_GROUP_ID"),
		existingProjectInGroup:    os.Getenv("TEST_EXISTING_PROJECT_IN_GROUP"),
		nonExistingProjectInGroup: os.Getenv("TEST_NON_EXISTING_PROJECT_IN_GROUP"),
		sshUrl:           os.Getenv("TEST_GITLAB_PROJECT_SSH_URL"),
		nonExistingGroup: os.Getenv("TEST_GITLAB_NON_EXISTING_GROUP"),
	}
}

func TestGitLab_CheckProjectExist_InvalidPass_ExistingProject(t *testing.T) {
	gitlab := createGitlab()
	client := GitLab{}
	_ = client.Init(gitlab.url, gitlab.user, "invalid")

	res, err := client.CheckProjectExist(gitlab.group, gitlab.existingProjectInGroup)

	if err == nil {
		t.Errorf("Actual: %v. Expected: error", &res)
	}
}

func TestGitLab_GetGroupIdByName_InvalidPass_ExistingGroup(t *testing.T) {
	gitlab := createGitlab()
	client := GitLab{}
	_ = client.Init(gitlab.url, gitlab.user, "invalid")

	res, err := client.GetGroupIdByName(gitlab.group)

	if err == nil {
		t.Errorf("Actual: %s. Expected: error", res)
	}
}

func TestGitLab_CreateProject_ValidPass_ExistingProject_ExistingGroup(t *testing.T) {
	gitlab := createGitlab()
	client := GitLab{}
	_ = client.Init(gitlab.url, gitlab.user, gitlab.pass)

	_, err := client.CreateProject(gitlab.group, gitlab.existingProjectInGroup)

	if err == nil {
		t.Errorf("Actual: created project. Expected: error")
	}
}

func TestGitLab_CreateProject_ValidPass_NonExistingProject_NonExistingGroup(t *testing.T) {
	gitlab := createGitlab()
	client := GitLab{}
	_ = client.Init(gitlab.url, gitlab.user, gitlab.pass)

	_, err := client.CreateProject("non-existing", "777777")

	if err == nil {
		t.Errorf("Actual: created project. Expected: error")
	}
}

func TestGitLab_CreateProject_InValidToken_NonExistingProject_ExistingGroup(t *testing.T) {
	gitlab := createGitlab()
	client := GitLab{}
	_ = client.Init(gitlab.url, gitlab.user, "invalid")

	id, err := client.CreateProject(gitlab.group, gitlab.nonExistingProjectInGroup)

	if err == nil {
		t.Errorf("Actual: %v. Expected: error", id)
	}
}

func TestGitLab_GetRepositorySshUrlNonExistingGroupPath(t *testing.T) {
	gitlab := createGitlab()
	client := GitLab{}
	_ = client.Init(gitlab.url, gitlab.user, gitlab.pass)

	sshUrl, err := client.GetRepositorySshUrl(gitlab.nonExistingGroup, gitlab.existingProjectInGroup)

	if err == nil {
		t.Errorf("Expected: error, actual response: %v", sshUrl)
		return
	}
}
