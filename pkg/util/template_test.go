package util

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
)

func TestCopyTemplate_HelmTemplates_ShouldPass(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	cf := &model.ConfigGoTemplating{
		Name:         "c-name",
		PlatformType: "kubernetes",
		Lang:         "go",
		DnsWildcard:  "mydomain.example.com",
		GitURL:       "https://example.com",
	}

	err := CopyTemplate(HelmChartDeploymentScriptType, testDir, "../../build", cf)
	require.NoError(t, err)

	chf := fmt.Sprintf("%v/deploy-templates/Chart.yaml", testDir)

	_, err = os.Stat(chf)
	if err != nil {
		t.Fatalf("failed to check test file")
	}

	// read the whole file at once
	b, err := os.ReadFile(chf)
	if err != nil {
		t.Fatalf("failed to read test file")
	}

	assert.Contains(t, string(b), "home: https://example.com")
}

func TestCopyTemplate_ShouldFailOnUnsupportedDeploymemntType(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	cf := &model.ConfigGoTemplating{}

	err := CopyTemplate("non-supported-deployment-type", testDir, "../../build", cf)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported deployment type")
}
