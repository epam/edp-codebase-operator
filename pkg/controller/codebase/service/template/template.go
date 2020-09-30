package template

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/platform"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
	"text/template"
)

var log = logf.Log.WithName("template")

func PrepareTemplates(client *coreV1Client.CoreV1Client, c v1alpha1.Codebase) error {
	log.Info("start preparing deploy templates", "codebase", c.Name)
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)

	cf, err := buildTemplateConfig(client, c)
	if err != nil {
		return err
	}

	if c.Spec.Type == util.Application {
		if err := util.CopyTemplate(*c.Spec.Framework, c.Spec.DeploymentScript, wd, *cf); err != nil {
			return errors.Wrapf(err, "an error has occurred while copying template for %v codebase", c.Name)
		}
	}

	if err := util.CopyPipelines(c.Spec.Type, util.PipelineTemplates, util.GetWorkDir(c.Name, c.Namespace)); err != nil {
		return errors.Wrapf(err, "an error has occurred while copying pipelines for %v codebase", c.Name)
	}

	if (strings.ToLower(c.Spec.Lang) == util.Javascript ||
		strings.ToLower(c.Spec.Lang) == strings.ToLower(util.LanguageGo)) &&
		c.Spec.Strategy != util.ImportStrategy {
		td := fmt.Sprintf("%v/%v", wd, "templates")
		if err := copySonarConfigs(td, *cf); err != nil {
			return err
		}
	}
	log.Info("end preparing deploy templates", "codebase", c.Name)
	return nil
}

func PrepareGitlabCITemplates(client *coreV1Client.CoreV1Client, c v1alpha1.Codebase) error {
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
	if err := util.CopyTemplate(*c.Spec.Framework, c.Spec.DeploymentScript, wd, *cf); err != nil {
		return errors.Wrapf(err, "an error has occurred while copying template for %v codebase", c.Name)
	}

	log.Info("end preparing deploy templates", "codebase", c.Name)
	return nil
}

func buildTemplateConfig(client *coreV1Client.CoreV1Client, c v1alpha1.Codebase) (*model.ConfigGoTemplating, error) {
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
	if c.Spec.Database != nil {
		cf.Database = c.Spec.Database
	}
	if c.Spec.Route != nil {
		cf.Route = c.Spec.Route
	}
	log.Info("end creating template config", "codebase name", c.Name)
	return &cf, nil
}

func copySonarConfigs(templateDirectory string, config model.ConfigGoTemplating) error {
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

	tmpl, err := template.New("sonar-project.properties.tmpl").
		ParseFiles("/usr/local/bin/templates/sonar/sonar-project.properties.tmpl")
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, config); err != nil {
		return errors.Wrapf(err, "couldn't render Sonar configs fo %v app: %v", config.Lang, config.Name)
	}
	log.Info("Sonar configs has been copied", "codebase name", config.Name)
	return nil
}
