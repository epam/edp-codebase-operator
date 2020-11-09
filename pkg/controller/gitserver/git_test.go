package gitserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	edpGerrit = "https://gerrit-oc-green-edp-cicd.delivery.aws.main.edp.projects.epam.com/public-repo-for-test"
)

func TestGitCheckPermissionsGerrit(t *testing.T) {
	gp := GitProvider{}
	assert.True(t, gp.CheckPermissions(edpGerrit, "", ""))
}

func TestBasicAuthEmptyUser(t *testing.T) {
	assert.Nil(t, basicAuth("", ""))
}

func TestBasicAuthFilledUser(t *testing.T) {
	ba := basicAuth("some", "some")
	assert.NotNil(t, ba)
	assert.Equal(t, "some", ba.Username)
	assert.Equal(t, "some", ba.Password)
}
