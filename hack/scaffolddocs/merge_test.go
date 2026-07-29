package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	envoyGuard = `{{- if eq .IngressController "envoy" }}`
	nginxGuard = `{{- if ne .IngressController "envoy" }}`
)

// doc builds generated documentation with the given value rows.
func doc(rows ...string) string {
	return strings.Join(append([]string{
		"# chart",
		"",
		"## Values",
		"",
		"| Key | Type | Default | Description |",
		"|-----|------|---------|-------------|",
	}, append(rows, "")...), "\n")
}

func TestParseReadme(t *testing.T) {
	t.Parallel()

	parsed, err := parseReadme(doc("| a | string | `\"1\"` |  |", "| b | int | `2` |  |"), envoyGuard)
	require.NoError(t, err)

	assert.Equal(t, envoyGuard, parsed.guard)
	assert.Equal(t, []string{"| a | string | `\"1\"` |  |", "| b | int | `2` |  |"}, parsed.rows)
	assert.Equal(t, "# chart", parsed.header[0])
	assert.Equal(t, []string{""}, parsed.footer)
}

func TestParseReadme_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		doc     string
		wantErr string
	}{
		{
			name:    "no values section",
			doc:     "# chart\n\nno values here\n",
			wantErr: "not found in generated documentation",
		},
		{
			name:    "no rows",
			doc:     doc(),
			wantErr: "values table has no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseReadme(tt.doc, envoyGuard)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestMergeReadmes_GuardsVariantSpecificRows(t *testing.T) {
	t.Parallel()

	// Rows exclusive to one variant must end up guarded, and the merged table must stay in the
	// alphanumeric order helm-docs sorts by - that ordering is what the drift was about.
	envoy, err := parseReadme(doc(
		"| fullnameOverride | string | `\"\"` |  |",
		"| httproute.enabled | bool | `false` |  |",
		"| image.tag | string | `\"\"` |  |",
	), envoyGuard)
	require.NoError(t, err)

	nginx, err := parseReadme(doc(
		"| fullnameOverride | string | `\"\"` |  |",
		"| image.tag | string | `\"\"` |  |",
		"| ingress.enabled | bool | `false` |  |",
	), nginxGuard)
	require.NoError(t, err)

	merged, err := mergeReadmes([]readme{envoy, nginx})
	require.NoError(t, err)

	assert.Contains(t, merged, strings.Join([]string{
		"| fullnameOverride | string | `\"\"` |  |",
		envoyGuard,
		"| httproute.enabled | bool | `false` |  |",
		guardEnd,
		"| image.tag | string | `\"\"` |  |",
		nginxGuard,
		"| ingress.enabled | bool | `false` |  |",
		guardEnd,
	}, "\n"))
}

func TestMergeReadmes_SingleVariantIsUnguarded(t *testing.T) {
	t.Parallel()

	only, err := parseReadme(doc("| replicaCount | int | `1` |  |"), "")
	require.NoError(t, err)

	merged, err := mergeReadmes([]readme{only})
	require.NoError(t, err)

	assert.NotContains(t, merged, guardEnd)
	assert.Contains(t, merged, "| replicaCount | int | `1` |  |")
}

func TestMergeReadmes_SameKeyDifferentValue(t *testing.T) {
	t.Parallel()

	// A key both variants document but with different content has to keep one row per variant,
	// each behind its own guard, so exactly one of them renders.
	envoy, err := parseReadme(doc("| service.port | int | `8080` |  |"), envoyGuard)
	require.NoError(t, err)

	nginx, err := parseReadme(doc("| service.port | int | `80` |  |"), nginxGuard)
	require.NoError(t, err)

	merged, err := mergeReadmes([]readme{envoy, nginx})
	require.NoError(t, err)

	assert.Contains(t, merged, strings.Join([]string{
		envoyGuard,
		"| service.port | int | `8080` |  |",
		guardEnd,
		nginxGuard,
		"| service.port | int | `80` |  |",
		guardEnd,
	}, "\n"))
}

func TestMergeReadmes_RejectsDifferencesOutsideTheTable(t *testing.T) {
	t.Parallel()

	envoy, err := parseReadme(doc("| a | int | `1` |  |"), envoyGuard)
	require.NoError(t, err)

	nginx, err := parseReadme(strings.Replace(doc("| a | int | `1` |  |"), "# chart", "# other", 1), nginxGuard)
	require.NoError(t, err)

	_, err = mergeReadmes([]readme{envoy, nginx})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "differ outside the values table")
}

func TestMergeReadmes_RejectsPartiallySharedRow(t *testing.T) {
	t.Parallel()

	// With three variants a row shared by two of them would need an OR of guards, which the
	// generator does not emit - it must fail loudly instead of dropping the row.
	shared := "| a | int | `1` |  |"

	first, err := parseReadme(doc(shared), envoyGuard)
	require.NoError(t, err)

	second, err := parseReadme(doc(shared), nginxGuard)
	require.NoError(t, err)

	third, err := parseReadme(doc("| b | int | `2` |  |"), "{{- if false }}")
	require.NoError(t, err)

	_, err = mergeReadmes([]readme{first, second, third})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a combined guard")
}

func TestMergeReadmes_NoVariants(t *testing.T) {
	t.Parallel()

	_, err := mergeReadmes(nil)
	require.Error(t, err)
}

func TestRowKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "image.tag", rowKey("| image.tag | string | `\"\"` | Overrides the tag. |"))
	assert.Equal(t, "imagePullSecrets[0].name", rowKey("| imagePullSecrets[0].name | string | `\"regcred\"` |  |"))
	assert.Equal(t, "not a row", rowKey("not a row"))
}
