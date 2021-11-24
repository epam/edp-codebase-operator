package bitbucket

import (
	"os"
	"testing"
)

type bitbucket struct {
	url                       string
	user                      string
	pass                      string
	token                     string
	group                     string
	groupId                   string
	existingProjectInGroup    string
	nonExistingProjectInGroup string
	sshLink                   string
}

func createBitbucket() bitbucket {
	return bitbucket{
		url:                       os.Getenv("TEST_BITBUCKET_URL"),
		user:                      os.Getenv("TEST_BITBUCKET_USERNAME"),
		pass:                      os.Getenv("TEST_BITBUCKET_PASSWORD"),
		token:                     os.Getenv("TEST_BITBUCKET_TOKEN"),
		group:                     os.Getenv("TEST_BITBUCKET_GROUP"),
		groupId:                   os.Getenv("TEST_BITBUCKET_GROUP_ID"),
		existingProjectInGroup:    os.Getenv("TEST_EXISTING_PROJECT_IN_GROUP"),
		nonExistingProjectInGroup: os.Getenv("TEST_NON_EXISTING_PROJECT_IN_GROUP"),
		sshLink:                   os.Getenv("TEST_BITBUCKET_SSH"),
	}
}

func TestBitBucket_CheckProjectExist_InvalidPass_ExistingProject(t *testing.T) {
	bitbucket := createBitbucket()
	client := BitBucket{}
	_ = client.Init(bitbucket.url, bitbucket.user, "invalid")

	res, err := client.CheckProjectExist(bitbucket.group, bitbucket.existingProjectInGroup)

	if err == nil {
		t.Errorf("Actual: %v. Expected: error", &res)
	}
}

func TestBitBucket_CreateProject_ValidPass_ExistingProject_ExistingGroup(t *testing.T) {
	bitbucket := createBitbucket()
	client := BitBucket{}
	_ = client.Init(bitbucket.url, bitbucket.user, bitbucket.pass)

	_, err := client.CreateProject(bitbucket.existingProjectInGroup, bitbucket.groupId)

	if err == nil {
		t.Errorf("Actual: created project. Expected: error")
	}
}

func TestBitBucket_CreateProject_ValidPass_NonExistingProject_NonExistingGroup(t *testing.T) {
	bitbucket := createBitbucket()
	client := BitBucket{}
	_ = client.Init(bitbucket.url, bitbucket.user, bitbucket.pass)

	_, err := client.CreateProject("fail", "fail")

	if err == nil {
		t.Errorf("Actual: created project. Expected: error")
	}
}

func TestBitBucket_CreateProject_InValidToken_NonExistingProject_ExistingGroup(t *testing.T) {
	bitbucket := createBitbucket()
	client := BitBucket{}
	_ = client.Init(bitbucket.url, bitbucket.user, "invalid")

	id, err := client.CreateProject(bitbucket.nonExistingProjectInGroup, bitbucket.groupId)

	if err == nil {
		t.Errorf("Actual: %v. Expected: error", id)
	}
}
