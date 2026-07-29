// Command scaffolddocs regenerates the README.tmpl files shipped with the application Helm
// chart scaffold.
//
// The templates are not hand-written. Each one is helm-docs output for the chart the operator
// actually scaffolds, with the codebase-specific values substituted back as Go template
// actions. Hand-transcribing that table is what let the documentation drift away from
// values.tmpl, which in turn failed the helm-docs task of every review pipeline.
//
// Run it through `make scaffold-docs`; `make validate-scaffold-docs` fails when the committed
// templates are stale.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// Values the scaffold is rendered with so helm-docs sees a syntactically valid chart. They are
// turned back into template actions afterwards, so none of them must be a prefix of another.
const (
	nameValue             = "scaffold-codebase-name"
	dnsWildcardValue      = "scaffold-dns-wildcard"
	gitURLValue           = "scaffold-git-url"
	frameworkValue        = "scaffold-framework"
	gatewayNameValue      = "scaffold-gateway-name"
	gatewayNamespaceValue = "scaffold-gateway-ns"
)

const chartDirName = "deploy-templates"

// placeholder pairs a value the operator substitutes at scaffold time with the template action
// that produces it.
type placeholder struct {
	value  string
	action string
}

func placeholders() []placeholder {
	return []placeholder{
		{value: nameValue, action: "{{.Name}}"},
		{value: dnsWildcardValue, action: "{{.DnsWildcard}}"},
		{value: gitURLValue, action: "{{.GitURL}}"},
		{value: frameworkValue, action: "{{.Framework}}"},
		{value: gatewayNameValue, action: "{{.GatewayName}}"},
		{value: gatewayNamespaceValue, action: "{{.GatewayNamespace}}"},
	}
}

// chartVariant is one rendering of a scaffold. Variants differ only in which values they expose,
// so their tables are merged back into a single template in which each variant-specific run of
// rows is wrapped in the guard that selects it.
type chartVariant struct {
	ingressController string
	guard             string
}

// chartPlatforms are the scaffold directories under build/templates/applications/helm-chart.
func chartPlatforms() []string {
	return []string{platform.K8S, platform.Openshift}
}

// chartVariants are rendered for every platform, because the ingress controller is selected
// independently of the platform. A platform whose scaffold ignores the selection renders both
// variants identically, every row is then documented by both, and the merged template comes out
// unguarded - so covering a platform that does not branch costs one helm-docs run and nothing
// else, while a platform that starts branching is documented correctly without touching this.
func chartVariants() []chartVariant {
	return []chartVariant{
		{
			ingressController: platform.IngressControllerEnvoy,
			guard:             fmt.Sprintf(`{{- if eq .IngressController %q }}`, platform.IngressControllerEnvoy),
		},
		{
			ingressController: platform.IngressControllerNginx,
			guard:             fmt.Sprintf(`{{- if ne .IngressController %q }}`, platform.IngressControllerEnvoy),
		},
	}
}

func main() {
	helmDocs := flag.String("helm-docs", filepath.Join("bin", "helm-docs"), "path to the helm-docs binary")
	assets := flag.String("assets", "build", "directory holding the scaffold templates")

	flag.Parse()

	if err := run(*helmDocs, *assets); err != nil {
		fmt.Fprintf(os.Stderr, "scaffolddocs: %v\n", err)
		os.Exit(1)
	}
}

func run(helmDocs, assets string) error {
	variants := chartVariants()

	for _, platformType := range chartPlatforms() {
		documented := make([]readme, 0, len(variants))

		for _, v := range variants {
			doc, err := documentVariant(helmDocs, assets, platformType, v)
			if err != nil {
				return fmt.Errorf("failed to document %s/%s: %w", platformType, v.ingressController, err)
			}

			documented = append(documented, doc)
		}

		merged, err := mergeReadmes(documented)
		if err != nil {
			return fmt.Errorf("failed to merge %s documentation: %w", platformType, err)
		}

		dest := filepath.Join(
			assets, "templates", "applications", util.HelmChartDeploymentScriptType, platformType, util.ReadmeTemplate,
		)

		if err = os.WriteFile(dest, []byte(merged), 0o644); err != nil {
			return fmt.Errorf("failed to write %q: %w", dest, err)
		}

		fmt.Println("generated", dest)
	}

	return nil
}

// documentVariant renders one variant of the scaffold and runs helm-docs over the result.
func documentVariant(helmDocs, assets, platformType string, v chartVariant) (readme, error) {
	workDir, err := os.MkdirTemp("", "scaffolddocs-")
	if err != nil {
		return readme{}, fmt.Errorf("failed to create work directory: %w", err)
	}

	defer func() { _ = os.RemoveAll(workDir) }()

	// Render through the operator's own code path, so the documentation cannot describe a
	// chart the operator would not produce.
	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

	err = util.CopyTemplate(ctx, util.HelmChartDeploymentScriptType, workDir, assets, config(platformType, v))
	if err != nil {
		return readme{}, fmt.Errorf("failed to render scaffold: %w", err)
	}

	if err = normalizeLineEndings(filepath.Join(workDir, chartDirName)); err != nil {
		return readme{}, err
	}

	// Invoked exactly like the review pipeline invokes it: from the repository root, against a
	// relative chart directory. helm-docs overwrites the README.md that CopyTemplate rendered
	// from the template being regenerated here, so the previous content cannot leak into it.
	cmd := exec.Command(helmDocs, "--chart-search-root", chartDirName)
	cmd.Dir = workDir

	if out, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		return readme{}, fmt.Errorf("failed to run helm-docs: %w: %s", cmdErr, out)
	}

	generated, err := os.ReadFile(filepath.Join(workDir, chartDirName, "README.md"))
	if err != nil {
		return readme{}, fmt.Errorf("failed to read generated documentation: %w", err)
	}

	return parseReadme(restorePlaceholders(strings.ReplaceAll(string(generated), "\r\n", "\n")), v.guard)
}

func config(platformType string, v chartVariant) *model.ConfigGoTemplating {
	return &model.ConfigGoTemplating{
		Name:              nameValue,
		PlatformType:      platformType,
		DnsWildcard:       dnsWildcardValue,
		GitURL:            gitURLValue,
		Framework:         frameworkValue,
		IngressController: v.ingressController,
		GatewayName:       gatewayNameValue,
		GatewayNamespace:  gatewayNamespaceValue,
	}
}

// restorePlaceholders turns the rendered values back into the template actions that produced
// them, so the generated file is a template again.
func restorePlaceholders(doc string) string {
	for _, p := range placeholders() {
		doc = strings.ReplaceAll(doc, p.value, p.action)
	}

	return doc
}

// normalizeLineEndings rewrites the rendered chart with LF endings. A Git checkout on Windows
// carries CRLF, which the markdown post-processing of helm-docs does not recognise, leaving
// blank runs and trailing spaces that make the generated template differ per platform.
func normalizeLineEndings(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", path, err)
		}

		normalized := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		if bytes.Equal(normalized, content) {
			return nil
		}

		if err = os.WriteFile(path, normalized, 0o644); err != nil {
			return fmt.Errorf("failed to write %q: %w", path, err)
		}

		return nil
	})
}
