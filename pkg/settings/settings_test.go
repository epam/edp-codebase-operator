package settings

import (
	"codebase-operator/models"
	"fmt"
	"testing"
)

func createAppSettings() models.CodebaseSettings {
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
	appSettings := createAppSettings()

	err := CreateSshConfig(appSettings)
	if err != nil {
		fmt.Println(err)
	}
	if err != nil {
		t.Error("Actual: error. Expected: true")
	}
}
