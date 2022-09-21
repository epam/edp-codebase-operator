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
	// ToDo: replace /tmp with t.TempDir()
	testDir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(testDir)
		require.NoError(t, err)
	}()

	cf := model.ConfigGoTemplating{
		Name:         "c-name",
		PlatformType: "kubernetes",
		Lang:         "go",
		DnsWildcard:  "mydomain.example.com",
		GitURL:       "https://example.com",
	}

	err = CopyTemplate(HelmChartDeploymentScriptType, testDir, "../../build", cf)
	require.NoError(t, err)

	chf := fmt.Sprintf("%v/deploy-templates/Chart.yaml", testDir)
	_, err = os.Stat(chf)
	if err != nil {
		t.Fatalf("unable to check test file")
	}
	// read the whole file at once
	b, err := os.ReadFile(chf)
	if err != nil {
		t.Fatalf("unable to read test file")
	}
	assert.Contains(t, string(b), "home: https://example.com")
}

func TestCopyTemplate_OpenShiftTemplates_ShouldPass(t *testing.T) {
	testDir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(testDir)
		require.NoError(t, err)
	}()

	cf := model.ConfigGoTemplating{
		Name:         "c-name",
		PlatformType: "kubernetes",
		Lang:         "go",
		DnsWildcard:  "mydomain.example.com",
		GitURL:       "https://example.com",
	}

	err = CopyTemplate("openshift-template", testDir, "../../build", cf)
	require.NoError(t, err)

	chf := fmt.Sprintf("%v/deploy-templates/c-name.yaml", testDir)
	_, err = os.Stat(chf)
	if err != nil {
		t.Fatalf("unable to check test file")
	}

	b, err := os.ReadFile(chf)
	if err != nil {
		t.Fatalf("unable to read test file")
	}
	assert.Contains(t, string(b), "description: Openshift template for Go application/service deploying")
}

func TestCopyTemplate_ShouldNotOverwriteExistingDeploymentTemaplates(t *testing.T) {
	testDir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(testDir)
		require.NoError(t, err)
	}()

	cf := model.ConfigGoTemplating{}
	if err = os.MkdirAll(fmt.Sprintf("%v/deploy-templates", testDir), 0775); err != nil {
		t.Fatal("Unable to create deploy-templates dir")
	}

	err = CopyTemplate("openshift-template", testDir, "../../build", cf)
	assert.NoError(t, err)
}

func TestCopyTemplate_ShouldFailOnUnsupportedDeploymemntType(t *testing.T) {
	testDir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(testDir)
		require.NoError(t, err)
	}()

	cf := model.ConfigGoTemplating{}

	err = CopyTemplate("non-supported-deployment-type", testDir, "../../build", cf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported deployment type")
}
