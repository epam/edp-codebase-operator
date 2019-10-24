package settings

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"testing"
)

func createCodebaseSettings() model.CodebaseSettings {
	return model.CodebaseSettings{
		ProjectVcsHostname: "gerrit",
		GerritKeyPath:      "/test/test",
		CicdNamespace:      "test",
		UserSettings: model.UserSettings{
			VcsIntegrationEnabled: true,
			VcsSshPort:            "22",
		},
		GerritSettings: model.GerritSettings{
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
