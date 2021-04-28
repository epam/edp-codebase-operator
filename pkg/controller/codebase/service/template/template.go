package template

import (
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
)

var log = ctrl.Log.WithName("template")

func PrepareTemplates(client client.Client, c v1alpha1.Codebase) error {
	log.Info("start preparing deploy templates", "codebase", c.Name)
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)

	cf, err := buildTemplateConfig(client, c)
	if err != nil {
		return err
	}

	if c.Spec.Type == util.Application {
		if err := util.CopyTemplate(c.Spec.DeploymentScript, wd, *cf); err != nil {
			return errors.Wrapf(err, "an error has occurred while copying template for %v codebase", c.Name)
		}
	}

	if err := util.CopyPipelines(c.Spec.Type, util.PipelineTemplates, util.GetWorkDir(c.Name, c.Namespace)); err != nil {
		return errors.Wrapf(err, "an error has occurred while copying pipelines for %v codebase", c.Name)
	}

	if c.Spec.Strategy != util.ImportStrategy {
		td := fmt.Sprintf("%v/%v", wd, "templates")
		if err := copySonarConfigs(td, *cf); err != nil {
			return err
		}
	}
	log.Info("end preparing deploy templates", "codebase", c.Name)
	return nil
}

func PrepareGitlabCITemplates(client client.Client, c v1alpha1.Codebase) error {
	log.Info("start preparing deploy templates", "codebase", c.Name)

	if c.Spec.Type != util.Application {
		log.Info("codebase is not application. skip copying templates", "name", c.Name)
		return nil
	}

	cf, err := buildTemplateConfig(client, c)
	if err != nil {
		return err
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	if err := util.CopyTemplate(c.Spec.DeploymentScript, wd, *cf); err != nil {
		return errors.Wrapf(err, "an error has occurred while copying template for %v codebase", c.Name)
	}

	log.Info("end preparing deploy templates", "codebase", c.Name)
	return nil
}

func buildTemplateConfig(client client.Client, c v1alpha1.Codebase) (*model.ConfigGoTemplating, error) {
	log.Info("start creating template config", "codebase name", c.Name)
	us, err := util.GetUserSettings(client, c.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "unable get user settings settings")
	}

	cf := model.ConfigGoTemplating{
		Name:         c.Name,
		PlatformType: platform.GetPlatformType(),
		Lang:         c.Spec.Lang,
		DnsWildcard:  us.DnsWildcard,
	}
	if c.Spec.Framework != nil {
		cf.Framework = *c.Spec.Framework
	}
	if c.Spec.Route != nil {
		cf.Route = c.Spec.Route
	}
	cf.GitURL, err = getProjectUrl(client, c.Spec, c.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "unable get project url")
	}

	log.Info("end creating template config", "codebase name", c.Name)
	return &cf, nil
}

func getProjectUrl (c client.Client, s v1alpha1.CodebaseSpec, n string) (string, error) {
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
			return "", errors.Wrap(err, "unable get git server")
		}
		return fmt.Sprintf("https://%v%v", gs.GitHost, *s.GitUrlPath), nil

	default:
		return "", errors.New("unable get project url, caused by the unsupported strategy")
	}
}

func copySonarConfigs(templateDirectory string, config model.ConfigGoTemplating) error {
	languagesForSonarTemplates := []string{util.LanguageJavascript, util.LanguagePython, util.LanguageGo}
	if !util.CheckElementInArray(languagesForSonarTemplates, strings.ToLower(config.Lang)) {
		return nil
	}

	sonarConfigPath := fmt.Sprintf("%v/%v/sonar-project.properties", templateDirectory, config.Name)
	log.Info("start copying sonar configs", "path", sonarConfigPath)
	if _, err := os.Stat(sonarConfigPath); err == nil {
		return nil
	}

	f, err := os.Create(sonarConfigPath)
	if err != nil {
		return err
	}
	defer f.Close()

	sonarTemplateName := fmt.Sprintf("%v-sonar-project.properties.tmpl", strings.ToLower(config.Lang))
	sonarTemplateFile := fmt.Sprintf("/usr/local/bin/templates/sonar/%v", sonarTemplateName)

	tmpl, err := template.New(sonarTemplateName).ParseFiles(sonarTemplateFile)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, config); err != nil {
		return errors.Wrapf(err, "couldn't render Sonar configs fo %v app: %v", config.Lang, config.Name)
	}
	log.Info("Sonar configs has been copied", "codebase name", config.Name)
	return nil
}
