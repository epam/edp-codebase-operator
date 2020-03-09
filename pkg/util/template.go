package util

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func CopyPipelines(codebaseType, src, dest string) error {
	log.Info("Start copying pipelines", "src", src, "target", dest)

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		if codebaseType == "autotests" && f.Name() == "build.groovy" {
			continue
		}

		input, err := ioutil.ReadFile(src + "/" + f.Name())
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dest, f.Name()), input, 0755)
		if err != nil {
			return err
		}
	}

	log.Info("Jenkins pipelines for codebase has been copied", "target", dest)

	return nil
}

func CopyHelmChartTemplates(deploymentScript, workDir string, config model.GerritConfigGoTemplating) error {
	log.Info("start handling Helm Chart templates", "codebase name", config.Name)
	templatesDest := fmt.Sprintf("%v/%v/%v/deploy-templates", workDir, "templates", config.Name)
	templateBasePath := fmt.Sprintf("/usr/local/bin/templates/applications/%v", deploymentScript)

	log.Info("Paths", "templatesDest", templatesDest, "templateBasePath", templateBasePath)

	err := CreateDirectory(templatesDest)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", templateBasePath)

	vf := fmt.Sprintf("%v/%v.yaml", templatesDest, "values")
	fv, err := os.Create(vf)
	if err != nil {
		return err
	}
	log.Info("file is created", "file", vf)

	cf := fmt.Sprintf("%v/%v.yaml", templatesDest, "Chart")
	fc, err := os.Create(cf)
	if err != nil {
		return err
	}
	log.Info("file is created", "file", cf)

	tf := fmt.Sprintf("%v/%v", templatesDest, TemplateFolder)
	err = CreateDirectory(tf)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", tf)

	tsf := fmt.Sprintf("%v/%v", templateBasePath, TemplateFolder)
	err = CopyFiles(tsf, tf)
	if err != nil {
		return err
	}
	log.Info("files were copied", "from", tsf, "to", tf)

	if err := renderTemplate(fv, fmt.Sprintf("%v/%v", templateBasePath, ChartValuesTemplate),
		ChartValuesTemplate, config); err != nil {
		return err
	}

	if err := renderTemplate(fc, fmt.Sprintf("%v/%v", templateBasePath, ChartTemplate), ChartTemplate, config); err != nil {
		return err
	}
	log.Info("end handling Helm Chart templates", "codebase name", config.Name)
	return nil
}

func CopyOpenshiftTemplate(framework, deploymentScript, workDir string, config model.GerritConfigGoTemplating) error {
	log.Info("start handling Openshift template", "codebase name", config.Name)
	templatesDest := fmt.Sprintf("%v/%v/%v/deploy-templates", workDir, "templates", config.Name)
	templateBasePath := fmt.Sprintf("/usr/local/bin/templates/applications/%v/%v",
		deploymentScript, strings.ToLower(config.Lang))
	templateName := fmt.Sprintf("%v.tmpl", strings.ToLower(config.Lang))

	log.Info("Paths", "templatesDest", templatesDest, "templateBasePath", templateBasePath,
		"templateName", templateName)

	err := CreateDirectory(templatesDest)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", templatesDest)

	fp := fmt.Sprintf("%v/%v.yaml", templatesDest, config.Name)
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	log.Info("file is created", "path", fp)

	if err := renderTemplate(f, fmt.Sprintf("%v/%v", templateBasePath, templateName), templateName, config); err != nil {
		return err
	}
	log.Info("end handling Openshift template", "codebase name", config.Name)
	return nil
}

func CopyTemplate(framework, deploymentScript, workDir string, cf model.GerritConfigGoTemplating) error {
	if deploymentScript == HelmChartDeploymentScriptType {
		return CopyHelmChartTemplates(deploymentScript, workDir, cf)
	}
	return CopyOpenshiftTemplate(framework, deploymentScript, workDir, cf)
}

func renderTemplate(file *os.File, templateBasePath, templateName string, config model.GerritConfigGoTemplating) error {
	log.Info("start rendering deploy template", "path", templateBasePath)

	tmpl, err := template.New(templateName).ParseFiles(templateBasePath)
	if err != nil {
		return errors.Wrap(err, "unable to parse codebase deploy template")
	}

	if err := tmpl.Execute(file, config); err != nil {
		return errors.Wrap(err, "unable to render codebase deploy template")
	}
	log.Info("template has been rendered", "codebase", config.Name)
	return nil
}
