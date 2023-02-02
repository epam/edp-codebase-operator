package gitlab

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
)

func TestGitLab_CheckProjectExist(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fprojects%2F%25252Fbackup%25252Ffake-name%3Fsimple=true/api/v4/projects/%252Fbackup%252Ffake-name?simple=true",
		httpmock.NewJsonResponderOrPanic(200, ""))

	g := &GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CheckProjectExist("/backup", "fake-name")

	assert.NoError(t, err)
	assert.True(t, *r)
}

func TestGitLab_CheckProjectExist_ShouldFailToAuth(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fprojects%2F%25252Fbackup%25252Ffake-name%3Fsimple=true/api/v4/projects/%252Fbackup%252Ffake-name?simple=true",
		httpmock.NewJsonResponderOrPanic(401, ""))

	g := &GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CheckProjectExist("/backup", "fake-name")

	assert.Error(t, err)
	assert.Nil(t, r)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGitLab_CheckProjectExist_ShouldFailToRequest(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	g := &GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CheckProjectExist("/backup", "fake-name")

	assert.Error(t, err)
	assert.Nil(t, r)
	assert.Contains(t, err.Error(), "failed to check project")
}

func TestGitLab_Init(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jr := map[string]string{
		"access_token":    "access",
		"ssh_url_to_repo": "ssh://url",
	}
	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	err := g.Init("https://gitlab.example.com", "user", "pass")

	assert.NoError(t, err)
}

func TestGitLab_Init_ShouldFail(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jr := map[string]string{
		"access_token":    "access",
		"ssh_url_to_repo": "ssh://url",
	}
	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	err := g.Init("//", "user", "pass")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get GitLab access token: Post")
}

func TestGitLab_Init_ShouldGetError(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	jr := "GiLab is Unavailable"

	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(500, &jr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	err := g.Init("https://gitlab.example.com", "user", "pass")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GiLab is Unavailable")
}

func TestGitLab_DeleteProject_ShouldPass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	httpmock.RegisterResponder("DELETE", "//%2Fapi%2Fv4%2Fprojects%2Ffake-project/api/v4/projects/fake-project",
		httpmock.NewJsonResponderOrPanic(200, ""))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	err := g.DeleteProject("fake-project")

	assert.NoError(t, err)
}

func TestGitLab_DeleteProject_ShouldPassWithError(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	httpmock.RegisterResponder("DELETE", "//%2Fapi%2Fv4%2Fprojects%2Ffake-project/api/v4/projects/fake-project",
		httpmock.NewJsonResponderOrPanic(404, ""))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	err := g.DeleteProject("fake-project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestGitLab_DeleteProject_ShouldFail(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	err := g.DeleteProject("fake-project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete project in GitLab")
}

func TestGitLab_CreateProject_ShouldPass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	gr := map[string]int{
		"id": 4,
	}

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fgroups%2Ffake-group%3Fsimple=true/api/v4/groups/fake-group?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &gr))

	httpmock.RegisterResponder("POST", "/api/v4/projects?name=fake-project&namespace_id=4",
		httpmock.NewJsonResponderOrPanic(200, &gr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CreateProject("fake-group", "fake-project")

	assert.NoError(t, err)
	assert.Equal(t, r, "4")
}

func TestGitLab_CreateProject_ShouldFailToGetGroupIdByName(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	gr := map[string]int{
		"id": 4,
	}

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fgroups%2Ffake-group%3Fsimple=true/api/v4/groups/fake-group?simple=true",
		httpmock.NewJsonResponderOrPanic(500, &gr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CreateProject("fake-group", "fake-project")

	assert.Error(t, err)
	assert.Equal(t, r, "")
}

func TestGitLab_CreateProject_ShouldFail(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	gr := map[string]int{
		"id": 4,
	}

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fgroups%2Ffake-group%3Fsimple=true/api/v4/groups/fake-group?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &gr))

	httpmock.RegisterResponder("POST", "/api/v4/projects?name=fake-project&namespace_id=4",
		httpmock.NewJsonResponderOrPanic(500, "FAILED"))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CreateProject("fake-group", "fake-project")

	assert.Error(t, err)
	assert.Equal(t, r, "")
	assert.Contains(t, err.Error(), "FAILED")
}

func TestGitLab_CreateProject_ShouldFailOnCreate(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	gr := map[string]int{
		"id": 4,
	}

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fgroups%2Ffake-group%3Fsimple=true/api/v4/groups/fake-group?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &gr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.CreateProject("fake-group", "fake-project")
	assert.Error(t, err)
	assert.Equal(t, r, "")
	assert.Contains(t, err.Error(), "failed to create project in GitLab")
}

func TestGitLab_GetGroupIdByName_ShouldFail(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.GetGroupIdByName("fake-group")
	assert.Error(t, err)
	assert.Equal(t, r, "")
	assert.Contains(t, err.Error(), "failed get repository group id")
}

func TestGitLab_GetRepositorySshUrl_ShouldPass(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	gr := map[string]string{
		"access_token":    "access",
		"ssh_url_to_repo": "ssh://url",
	}
	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fprojects%2Ffake-group%25252Ffake-project%3Fsimple=true/api/v4/projects/fake-group%252Ffake-project?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &gr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	r, err := g.GetRepositorySshUrl("fake-group", "fake-project")
	assert.NoError(t, err)
	assert.Equal(t, r, "ssh://url")
}

func TestGitLab_GetRepositorySshUrl_ShouldFailToExecute(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	_, err := g.GetRepositorySshUrl("fake-group", "fake-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed get repository SSH URL")
}

func TestGitLab_GetRepositorySshUrl_ShouldNotFound(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fprojects%2Ffake-group%25252Ffake-project%3Fsimple=true/api/v4/projects/fake-group%252Ffake-project?simple=true",
		httpmock.NewJsonResponderOrPanic(404, "not found"))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	_, err := g.GetRepositorySshUrl("fake-group", "fake-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error has received by get project path")
}

func TestGitLab_GetRepositorySshUrl_ShouldNotFoundSSHUrlInResponse(t *testing.T) {
	httpmock.Reset()
	httpmock.Activate()

	gr := map[string]string{
		"access_token": "access",
	}
	httpmock.RegisterResponder("GET", "//%2Fapi%2Fv4%2Fprojects%2Ffake-group%25252Ffake-project%3Fsimple=true/api/v4/projects/fake-group%252Ffake-project?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &gr))

	g := GitLab{
		Client: *resty.DefaultClient,
	}

	_, err := g.GetRepositorySshUrl("fake-group", "fake-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSH URL is not presented in the response by group path: fake-group, project name: fake-project")
}
