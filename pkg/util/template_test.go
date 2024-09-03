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

func TestCopyHelmChartTemplates(t *testing.T) {
	tmp := t.TempDir()

	type args struct {
		deploymentScript string
		templatesDest    string
		assetsDir        string
		config           *model.ConfigGoTemplating
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "invalid assets dir",
			args: args{
				deploymentScript: HelmChartDeploymentScriptType,
				templatesDest:    tmp,
				assetsDir:        "",
				config:           &model.ConfigGoTemplating{},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to read a content of directory")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyHelmChartTemplates(
				tt.args.deploymentScript,
				tt.args.templatesDest,
				tt.args.assetsDir,
				tt.args.config,
			)

			tt.wantErr(t, err)
		})
	}
}

func TestCopyTemplate_RPMPackage_ShouldPass(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	cf := &model.ConfigGoTemplating{
		Name:         "c-name",
		PlatformType: "kubernetes",
		Lang:         "java",
		DnsWildcard:  "mydomain.example.com",
		GitURL:       "https://example.com",
	}

	err := CopyTemplate(RpmPackageDeploymentScriptType, testDir, "../../build", cf)
	require.NoError(t, err)

	chf := fmt.Sprintf("%v/c-name.spec", testDir)

	_, err = os.Stat(chf)
	if err != nil {
		t.Fatalf("failed to check test file")
	}

	// read the whole file at once
	b, err := os.ReadFile(chf)
	if err != nil {
		t.Fatalf("failed to read test file")
	}

	assert.Contains(t, string(b), "URL:            https://example.com")
}

func TestCopyRpmPackageTemplates(t *testing.T) {
	tmp := t.TempDir()

	type args struct {
		deploymentScript string
		templatesDest    string
		assetsDir        string
		config           *model.ConfigGoTemplating
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "invalid assets dir",
			args: args{
				deploymentScript: HelmChartDeploymentScriptType,
				templatesDest:    tmp,
				assetsDir:        "",
				config:           &model.ConfigGoTemplating{},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to parse codebase deploy template")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyRpmPackageTemplates(
				tt.args.templatesDest,
				tt.args.assetsDir,
				tt.args.config,
			)

			tt.wantErr(t, err)
		})
	}
}
