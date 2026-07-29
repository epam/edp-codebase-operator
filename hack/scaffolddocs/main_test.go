package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// fieldAction matches a Go template action that substitutes a single config field.
var fieldAction = regexp.MustCompile(`\{\{\s*\.([A-Za-z]+)\s*\}\}`)

// TestPlaceholders_CoverEveryTemplateAction fails when a scaffold template gains a field the
// generator does not supply. Such a field renders empty, which silently drops the row
// documenting it from the generated documentation instead of failing the generator.
func TestPlaceholders_CoverEveryTemplateAction(t *testing.T) {
	t.Parallel()

	// Scoped to the scaffold the generator renders: the other deployment scripts are templated
	// with a different config and are none of its business.
	templates, err := filepath.Glob(filepath.Join(
		"..", "..", "build", "templates", "applications", util.HelmChartDeploymentScriptType, "*", "*.tmpl",
	))
	require.NoError(t, err)
	require.NotEmpty(t, templates)

	supplied := make(map[string]bool, len(placeholders()))

	for _, p := range placeholders() {
		field := fieldAction.FindStringSubmatch(p.action)
		require.Len(t, field, 2, "placeholder action %q does not substitute a field", p.action)

		supplied[field[1]] = true
	}

	for _, path := range templates {
		// The README template is the generator's own output, described by the others.
		if filepath.Base(path) == util.ReadmeTemplate {
			continue
		}

		content, err := os.ReadFile(path)
		require.NoError(t, err)

		for _, match := range fieldAction.FindAllStringSubmatch(string(content), -1) {
			assert.True(t, supplied[match[1]],
				"%s substitutes {{.%s}}, which no placeholder supplies", path, match[1])
		}
	}
}

func TestPlaceholders_AreSubstitutableIndependently(t *testing.T) {
	t.Parallel()

	// Placeholders are substituted one after another over the whole document, so a placeholder
	// contained in another one would corrupt the longer one.
	for i, outer := range placeholders() {
		for j, inner := range placeholders() {
			if i == j {
				continue
			}

			assert.NotContains(t, outer.value, inner.value,
				"placeholder %q contains %q", outer.value, inner.value)
		}
	}
}

func TestRestorePlaceholders(t *testing.T) {
	t.Parallel()

	restored := restorePlaceholders(strings.Join([]string{
		"# " + nameValue,
		"**Homepage:** <" + gitURLValue + ">",
		"| httproute.dnsWildcard | string | `\"" + dnsWildcardValue + "\"` |  |",
		"| httproute.gateway.name | string | `\"" + gatewayNameValue + "\"` |  |",
		"| httproute.gateway.namespace | string | `\"" + gatewayNamespaceValue + "\"` |  |",
		"| framework | string | `\"" + frameworkValue + "\"` |  |",
	}, "\n"))

	assert.NotContains(t, restored, "scaffold-", "every placeholder must be turned back into an action")
	assert.Contains(t, restored, "# {{.Name}}")
	assert.Contains(t, restored, "<{{.GitURL}}>")
	assert.Contains(t, restored, "`\"{{.DnsWildcard}}\"`")
	assert.Contains(t, restored, "`\"{{.GatewayName}}\"`")
	assert.Contains(t, restored, "`\"{{.GatewayNamespace}}\"`")
	assert.Contains(t, restored, "`\"{{.Framework}}\"`")
}

func TestConfig_FillsEveryPlaceholder(t *testing.T) {
	t.Parallel()

	// Every placeholder has to be supplied to the scaffold, otherwise its values render empty
	// and the rows documenting them silently disappear from the generated template.
	cfg := config(platform.K8S, chartVariants()[0])

	supplied := []string{
		cfg.Name, cfg.DnsWildcard, cfg.GitURL, cfg.Framework, cfg.GatewayName, cfg.GatewayNamespace,
	}

	for _, p := range placeholders() {
		assert.Contains(t, supplied, p.value, "placeholder %q is not supplied by config()", p.value)
	}
}

// TestChartVariants_CoverBothControllersWithDistinctGuards keeps the generator's variants a
// partition of what the scaffold can branch on. A controller left out would be documented by
// whichever variant did run, silently, because the generated template would still be exactly
// what the generator produces and the CI check compares against that.
func TestChartVariants_CoverBothControllersWithDistinctGuards(t *testing.T) {
	t.Parallel()

	variants := chartVariants()
	require.Len(t, variants, 2, "both ingress controllers have to be rendered")

	controllers := make([]string, 0, len(variants))
	guards := make(map[string]bool, len(variants))

	for _, v := range variants {
		require.NotEmpty(t, v.guard, "variant %q is unguarded", v.ingressController)
		require.False(t, guards[v.guard], "guard %q is reused", v.guard)

		controllers = append(controllers, v.ingressController)
		guards[v.guard] = true
	}

	assert.Contains(t, controllers, platform.IngressControllerEnvoy)
	assert.Contains(t, controllers, platform.IngressControllerNginx)
}

// TestChartPlatforms_CoverEveryScaffoldDirectory fails when a scaffold directory is added without
// registering it, which would otherwise leave its README.tmpl hand-maintained and unvalidated -
// exactly the state this generator exists to remove.
func TestChartPlatforms_CoverEveryScaffoldDirectory(t *testing.T) {
	t.Parallel()

	scaffolds, err := filepath.Glob(filepath.Join(
		"..", "..", "build", "templates", "applications", util.HelmChartDeploymentScriptType, "*",
	))
	require.NoError(t, err)
	require.NotEmpty(t, scaffolds)

	for _, path := range scaffolds {
		info, err := os.Stat(path)
		require.NoError(t, err)

		if !info.IsDir() {
			continue
		}

		assert.Contains(t, chartPlatforms(), filepath.Base(path),
			"scaffold %q is not registered, so its documentation is never generated", path)
	}
}
