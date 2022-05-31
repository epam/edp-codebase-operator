package vcs

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateVCSClient_ShouldPassForBitbucket(t *testing.T) {
	vcs, err := CreateVCSClient(model.BitBucket, "url", "user", "pass")
	assert.NoError(t, err)
	assert.NotNil(t, vcs)
}

func TestCreateVCSClient_ShouldFailForUnsupportedVCS(t *testing.T) {
	_, err := CreateVCSClient("custom", "url", "user", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid VCS tool. Currently we do not support custom")
}

func TestCreateProjectInVcs_ShouldPassToCreateRepo(t *testing.T) {
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "vcs-autouser-codebase-fake-name-temp",
			Namespace: "fake-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build()

	httpmock.Reset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?private_token=pass&simple=true",
		httpmock.NewStringResponder(200, ""))

	jr := map[string]string{
		"access_token":    "access",
		"ssh_url_to_repo": "ssh://url",
	}
	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	us := model.UserSettings{
		VcsGroupNameUrl: "https://gitlab.example.com/backup",
		VcsToolName:     model.GitLab,
		VcsSshPort:      "22",
	}

	err := СreateProjectInVcs(fakeCl, &us, "fake-name", "fake-namespace")
	assert.NoError(t, err)
}

func TestСreateProjectInVcs_ShouldFailToInitGitlabClient(t *testing.T) {
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "vcs-autouser-codebase-fake-name-temp",
			Namespace: "fake-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build()

	httpmock.Reset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?private_token=pass&simple=true",
		httpmock.NewStringResponder(200, ""))

	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(500, ""))

	us := model.UserSettings{
		VcsGroupNameUrl: "https://gitlab.example.com/backup",
		VcsToolName:     model.GitLab,
		VcsSshPort:      "22",
	}

	err := СreateProjectInVcs(fakeCl, &us, "fake-name", "fake-namespace")
	assert.Error(t, err)
}

func TestСreateProjectInVcs_ShouldPassOnExistingProject(t *testing.T) {
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "vcs-autouser-codebase-fake-name-temp",
			Namespace: "fake-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build()

	httpmock.Reset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?private_token=pass&simple=true",
		httpmock.NewStringResponder(200, ""))

	jr := map[string]string{
		"access_token":    "access",
		"ssh_url_to_repo": "ssh://url",
	}
	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	us := model.UserSettings{
		VcsGroupNameUrl: "https://gitlab.example.com/backup",
		VcsToolName:     model.GitLab,
		VcsSshPort:      "22",
	}

	err := СreateProjectInVcs(fakeCl, &us, "fake-name", "fake-namespace")
	assert.NoError(t, err)
}

func TestGetVcsConfig_ShouldFailToGetSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()
	us := model.UserSettings{}
	_, err := GetVcsConfig(fakeCl, &us, "codebaseName", "fake-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to get secret")
}

func TestGetVcsConfig_ShouldFailToUrl(t *testing.T) {

	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "vcs-autouser-codebase-codebaseName-temp",
			Namespace: "fake-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build()
	us := model.UserSettings{
		VcsGroupNameUrl: ":",
	}
	_, err := GetVcsConfig(fakeCl, &us, "codebaseName", "fake-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing protocol scheme")
}
