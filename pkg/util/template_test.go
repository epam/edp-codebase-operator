package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestCopyPipelines_ShouldFailWhenReadFiles(t *testing.T) {
	err := CopyPipelines("application", "/tmp/1", "/tmp/2")
	assert.Error(t, err)
}

func TestCopyPipelines_ShouldPass(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)

	err = CopyPipelines("application", "../../build/pipelines", testDir)
	assert.NoError(t, err)
}

func TestCopyPipelines_ShouldSkipToCopyExistingFiles(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)

	f, err := os.Create(fmt.Sprintf("%v/build.groovy", testDir))
	if err != nil {
		t.Fatalf("unable to create file for testing")
	}

	err = CopyPipelines("application", "../../build/pipelines", testDir)
	assert.NoError(t, err)
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("unable to get created file for testing")
	}
	assert.Equal(t, int64(0), fi.Size())
}

func TestCopyPipelines_ShouldNotCreateBuildGroovyForAutotests(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)

	err = CopyPipelines("autotests", "../../build/pipelines", testDir)
	assert.NoError(t, err)
	_, err = os.Stat(fmt.Sprintf("%v/build.groovy", testDir))
	assert.True(t, os.IsNotExist(err))
}

func TestCopyTemplate_HelmTemplates_ShouldPass(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)
	cf := model.ConfigGoTemplating{
		Name:         "c-name",
		PlatformType: "kubernetes",
		Lang:         "go",
		DnsWildcard:  "mydomain.example.com",
		GitURL:       "https://example.com",
	}

	err = CopyTemplate(HelmChartDeploymentScriptType, testDir, "../../build", cf)
	assert.NoError(t, err)

	chf := fmt.Sprintf("%v/deploy-templates/Chart.yaml", testDir)
	_, err = os.Stat(chf)
	if err != nil {
		t.Fatalf("unable to check test file")
	}
	// read the whole file at once
	b, err := ioutil.ReadFile(chf)
	if err != nil {
		t.Fatalf("unable to read test file")
	}
	assert.Contains(t, string(b), "home: https://example.com")
}

func TestCopyTemplate_OpenShiftTemplates_ShouldPass(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)
	cf := model.ConfigGoTemplating{
		Name:         "c-name",
		PlatformType: "kubernetes",
		Lang:         "go",
		DnsWildcard:  "mydomain.example.com",
		GitURL:       "https://example.com",
	}

	err = CopyTemplate("openshift-template", testDir, "../../build", cf)
	assert.NoError(t, err)

	chf := fmt.Sprintf("%v/deploy-templates/c-name.yaml", testDir)
	_, err = os.Stat(chf)
	if err != nil {
		t.Fatalf("unable to check test file")
	}

	b, err := ioutil.ReadFile(chf)
	if err != nil {
		t.Fatalf("unable to read test file")
	}
	assert.Contains(t, string(b), "description: Openshift template for Go application/service deploying")
}

func TestCopyTemplate_ShouldNotOverwriteExistingDeploymentTemaplates(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)
	cf := model.ConfigGoTemplating{}
	os.MkdirAll(fmt.Sprintf("%v/deploy-templates", testDir), 0775)

	err = CopyTemplate("openshift-template", testDir, "../../build", cf)
	assert.NoError(t, err)
}

func TestCopyTemplate_ShouldFailOnUnsupportedDeploymemntType(t *testing.T) {
	testDir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(testDir)
	cf := model.ConfigGoTemplating{}

	err = CopyTemplate("non-supported-deployment-type", testDir, "../../build", cf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported deployment type")
}
