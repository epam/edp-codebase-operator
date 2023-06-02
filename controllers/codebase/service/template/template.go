package template

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var (
	languagesForSonarTemplates = []string{util.LanguageJavascript, util.LanguagePython, util.LanguageGo}
)

func PrepareTemplates(ctx context.Context, c client.Client, cb *codebaseApi.Codebase, workDir string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start preparing deploy templates")

	cf, err := buildTemplateConfig(ctx, c, cb)
	if err != nil {
		return err
	}

	assetsDir, err := util.GetAssetsDir()
	if err != nil {
		return fmt.Errorf("failed to get assets dir: %w", err)
	}

	if err := util.CopyTemplate(cb.Spec.DeploymentScript, workDir, assetsDir, cf); err != nil {
		return fmt.Errorf("failed to copy template for %v codebase: %w", cb.Name, err)
	}

	if err := copySonarConfigs(ctx, workDir, assetsDir, cf); err != nil {
		return err
	}

	log.Info("End preparing deploy templates")

	return nil
}

func buildTemplateConfig(ctx context.Context, c client.Client, cb *codebaseApi.Codebase) (*model.ConfigGoTemplating, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating template config")

	us, err := util.GetUserSettings(c, cb.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings settings: %w", err)
	}

	cf := model.ConfigGoTemplating{
		Name:         cb.Name,
		PlatformType: platform.GetPlatformType(),
		Lang:         cb.Spec.Lang,
		DnsWildcard:  us.DnsWildcard,
		Framework:    cb.Spec.Framework,
	}

	cf.GitURL, err = getProjectUrl(c, &cb.Spec, cb.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get project url: %w", err)
	}

	log.Info("End creating template config")

	return &cf, nil
}

func getProjectUrl(c client.Client, s *codebaseApi.CodebaseSpec, n string) (string, error) {
	switch s.Strategy {
	case "create":
		p := util.BuildRepoUrl(s)
		return p, nil

	case "clone":
		p := s.Repository.Url
		return p, nil

	case "import":
		gs, err := util.GetGitServer(c, s.GitServer, n)
		if err != nil {
			return "", fmt.Errorf("failed to get git server: %w", err)
		}

		return fmt.Sprintf("https://%v%v", gs.GitHost, *s.GitUrlPath), nil

	default:
		return "", errors.New("failed to get project url, caused by the unsupported strategy")
	}
}

// Copy sonar configurations for JavaScript, Python, Go
// It expects workDir - work dir, which contains codebase; td - template dir, which contains sonar.property file
// It returns error in case of issue.
func copySonarConfigs(ctx context.Context, workDir, td string, config *model.ConfigGoTemplating) (err error) {
	log := ctrl.LoggerFrom(ctx)

	if !slices.Contains(languagesForSonarTemplates, strings.ToLower(config.Lang)) {
		return
	}

	sonarConfigPath := fmt.Sprintf("%v/sonar-project.properties", workDir)
	log.Info("Start copying sonar configs", "path", sonarConfigPath)

	_, statErr := os.Stat(sonarConfigPath)
	if statErr == nil {
		log.Info("Sonar configs already exist")

		return nil
	}

	f, err := os.Create(sonarConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create sonar config file: %w", err)
	}

	defer util.CloseWithErrorCapture(&err, f, "failed to close sonar config file")

	sonarTemplateName := fmt.Sprintf("%v-sonar-project.properties.tmpl", strings.ToLower(config.Lang))
	sonarTemplateFile := fmt.Sprintf("%v/templates/sonar/%v", td, sonarTemplateName)

	tmpl, err := template.New(sonarTemplateName).ParseFiles(sonarTemplateFile)
	if err != nil {
		return fmt.Errorf("failed to parse sonar template file: %w", err)
	}

	err = tmpl.Execute(f, config)
	if err != nil {
		return fmt.Errorf("failed to render Sonar configs fo %v app: %v: %w", config.Lang, config.Name, err)
	}

	log.Info("Sonar configs has been copied")

	return
}
