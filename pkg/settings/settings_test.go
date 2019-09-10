package settings

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/models"
	"testing"
)

func createCodebaseSettings() models.CodebaseSettings {
	return models.CodebaseSettings{
		ProjectVcsHostname: "gerrit",
		GerritKeyPath:      "/test/test",
		CicdNamespace:      "test",
		UserSettings: models.UserSettings{
			VcsIntegrationEnabled: true,
			VcsSshPort:            "22",
		},
		GerritSettings: models.GerritSettings{
			SshPort: 22442,
		},
	}
}

func TestExampleSettings_CreateSshConfig(t *testing.T) {
	codebaseSettings := createCodebaseSettings()

	err := CreateSshConfig(codebaseSettings)
	if err != nil {
		fmt.Println(err)
	}
	if err != nil {
		t.Error("Actual: error. Expected: true")
	}
}
