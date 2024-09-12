package util

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

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

	err := CopyTemplate(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		HelmChartDeploymentScriptType,
		testDir,
		"../../build",
		cf,
	)
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

func TestCopyTemplate_HelmTemplates_DirectoryExists(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	_, err := os.Create(fmt.Sprintf("%v/deploy-templates", testDir))
	require.NoError(t, err)

	err = CopyTemplate(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		HelmChartDeploymentScriptType,
		testDir,
		"../../build",
		&model.ConfigGoTemplating{},
	)
	require.NoError(t, err)
}

func TestCopyTemplate_ShouldFailOnUnsupportedDeploymemntType(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	cf := &model.ConfigGoTemplating{}

	err := CopyTemplate(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		"non-supported-deployment-type",
		testDir,
		"../../build",
		cf,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported deployment type")
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

	err := CopyTemplate(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		RpmPackageDeploymentScriptType,
		testDir,
		"../../build",
		cf,
	)
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
	t.Parallel()

	type args struct {
		templatesDest func(t *testing.T) string
		config        *model.ConfigGoTemplating
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
		want    func(t *testing.T, dest string)
	}{
		{
			name: "templates created successfully",
			args: args{
				templatesDest: func(t *testing.T) string {
					return t.TempDir()
				},
				config: &model.ConfigGoTemplating{
					Lang: "java",
					Name: "test-application",
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, dest string) {
				_, err := os.Stat(fmt.Sprintf("%s/Makefile", dest))
				require.NoError(t, err)

				_, err = os.Stat(fmt.Sprintf("%s/Makefile.kuberocketci", dest))
				require.True(t, os.IsNotExist(err))

				_, err = os.Stat(fmt.Sprintf("%s/test-application.spec", dest))
				require.NoError(t, err)

				b, err := os.ReadFile(fmt.Sprintf("%s/test-application.spec", dest))
				require.Contains(t, string(b), "test-application")

				b, err = os.ReadFile(fmt.Sprintf("%s/test-application.service", dest))
				require.Contains(t, string(b), "test-application")

				_, err = os.Stat(fmt.Sprintf("%s/test-application.service", dest))
				require.NoError(t, err)

				_, err = os.Stat(fmt.Sprintf("%s/.rpmlintrc.toml", dest))
				require.NoError(t, err)
			},
		},
		{
			name: "templates with not supported lang created successfully",
			args: args{
				templatesDest: func(t *testing.T) string {
					return t.TempDir()
				},
				config: &model.ConfigGoTemplating{
					Lang: "not-supported",
					Name: "test",
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, dest string) {
				_, err := os.Stat(fmt.Sprintf("%s/Makefile", dest))
				require.NoError(t, err)
				_, err = os.Stat(fmt.Sprintf("%s/test.spec", dest))
				require.NoError(t, err)
				_, err = os.Stat(fmt.Sprintf("%s/test.service", dest))
				require.NoError(t, err)
				_, err = os.Stat(fmt.Sprintf("%s/.rpmlintrc.toml", dest))
				require.NoError(t, err)
			},
		},
		{
			name: "makefile already exists",
			args: args{
				templatesDest: func(t *testing.T) string {
					d := t.TempDir()

					f, err := os.Create(fmt.Sprintf("%s/Makefile", d))
					require.NoError(t, err)
					require.NoError(t, f.Close())

					return d
				},
				config: &model.ConfigGoTemplating{
					Lang: "java",
					Name: "test",
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, dest string) {
				_, err := os.Stat(fmt.Sprintf("%s/Makefile.kuberocketci", dest))
				require.NoError(t, err)
				_, err = os.Stat(fmt.Sprintf("%s/test.spec", dest))
				require.NoError(t, err)
				_, err = os.Stat(fmt.Sprintf("%s/test.service", dest))
				require.NoError(t, err)
				_, err = os.Stat(fmt.Sprintf("%s/.rpmlintrc.toml", dest))
				require.NoError(t, err)
			},
		},
		{
			name: "failed to create destination file",
			args: args{
				templatesDest: func(t *testing.T) string {
					return "invalid-dir"
				},
				config: &model.ConfigGoTemplating{},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create destination file")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dest := tt.args.templatesDest(t)

			err := CopyRpmPackageTemplates(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				dest,
				"../../build",
				tt.args.config,
			)

			tt.wantErr(t, err)
			if tt.want != nil {
				tt.want(t, dest)
			}
		})
	}
}
