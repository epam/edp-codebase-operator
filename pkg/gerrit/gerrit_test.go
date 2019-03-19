package gerrit

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateGerritReplicaConfig(t *testing.T) {
	templatePath := os.Getenv("GERRIT_TEMPLATE_PATH")
	rc, err := generateReplicationConfig(templatePath, ReplicationConfigTemplateName, ReplicationConfigParams{
		Name:      "test",
		VcsSshUrl: "git@git",
	})

	if err != nil {
		t.Error(err)
		return
	}
	if !strings.Contains(rc, "test") {
		t.Errorf("Expected: contains name, actual: %v", rc)
	}
}
