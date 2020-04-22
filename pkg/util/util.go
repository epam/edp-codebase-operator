package util

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	errWrap "github.com/pkg/errors"
	"io/ioutil"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
	"text/template"
)

var log = logf.Log.WithName("util-service")

func CreateDirectory(path string) error {
	log.Info("Creating directory for oc templates", "path", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}

	log.Info("Directory has been created", "path", path)

	return nil
}

func CopyPipelines(codebaseType string, src, pipelineDestination string) error {
	log.Info("Start copying pipelines", "src", src, "target", pipelineDestination)

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

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", pipelineDestination, f.Name()), input, 0755)
		if err != nil {
			return err
		}
	}

	log.Info("Jenkins pipelines for codebase has been copied", "target", pipelineDestination)

	return nil
}

func CopyHelmChartTemplates(config model.GerritConfigGoTemplating) error {
	templatesDest := createTemplateFolderPath(config.CodebaseSettings.WorkDir, config.CodebaseSettings.Name,
		config.CodebaseSettings.DeploymentScript)
	templateBasePath := fmt.Sprintf("/usr/local/bin/templates/applications/%v",
		config.CodebaseSettings.DeploymentScript)

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

	err = renderTemplate(fv, fmt.Sprintf("%v/%v", templateBasePath, ChartValuesTemplate),
		ChartValuesTemplate, config)
	if err != nil {
		return err
	}

	err = renderTemplate(fc, fmt.Sprintf("%v/%v", templateBasePath, ChartTemplate), ChartTemplate, config)
	if err != nil {
		return err
	}

	return nil
}

func CopyOpenshiftTemplate(config model.GerritConfigGoTemplating) error {
	templatesDest := createTemplateFolderPath(config.CodebaseSettings.WorkDir, config.CodebaseSettings.Name,
		config.CodebaseSettings.DeploymentScript)
	templateBasePath := fmt.Sprintf("/usr/local/bin/templates/applications/%v/%v",
		config.CodebaseSettings.DeploymentScript, strings.ToLower(config.Lang))
	templateName := fmt.Sprintf("%v.tmpl", strings.ToLower(*config.Framework))

	log.Info("Paths", "templatesDest", templatesDest, "templateBasePath", templateBasePath,
		"templateName", templateName)

	err := CreateDirectory(templatesDest)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", templatesDest)

	fp := fmt.Sprintf("%v/%v.yaml", templatesDest, config.CodebaseSettings.Name)
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	log.Info("file is created", "path", fp)

	return renderTemplate(f, fmt.Sprintf("%v/%v", templateBasePath, templateName), templateName, config)
}

func CopyTemplate(config model.GerritConfigGoTemplating) error {
	if config.CodebaseSettings.DeploymentScript == HelmChartDeploymentScriptType {
		return CopyHelmChartTemplates(config)
	}
	return CopyOpenshiftTemplate(config)
}

func renderTemplate(file *os.File, templateBasePath, templateName string, config model.GerritConfigGoTemplating) error {
	log.Info("Start rendering Helm Chart template", "path", templateBasePath)

	tmpl, err := template.New(templateName).ParseFiles(templateBasePath)
	if err != nil {
		return errWrap.Wrap(err, "unable to parse codebase deploy template")
	}

	err = tmpl.Execute(file, config)
	if err != nil {
		return errWrap.Wrap(err, "unable to render codebase deploy template")
	}

	log.Info("Helm Chart template has been rendered", "codebase", config.CodebaseSettings.Name)

	return nil
}

func createTemplateFolderPath(workDir, name, deploymentScriptType string) string {
	if deploymentScriptType == OpenshiftTemplate {
		return fmt.Sprintf("%v/%v/%v/deploy-templates", workDir, OcTemplatesFolder, name)
	}
	return fmt.Sprintf("%v/%v/%v/deploy-templates", workDir, HelmChartTemplatesFolder, name)
}

func CopyFiles(src, dest string) error {
	log.Info("Start copying files", "src", src, "dest", dest)

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		input, err := ioutil.ReadFile(src + "/" + f.Name())
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dest, f.Name()), input, 0755)
		if err != nil {
			return err
		}
	}

	log.Info("Files have been copied", "dest", dest)

	return nil
}

func CopyFile(src, dest string) error {
	log.Info("Start copying file", "src", src, "dest", dest)

	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, input, 0755)
	if err != nil {
		return err
	}

	log.Info("File has been copied", "dest", dest)

	return nil
}

func RemoveDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return errWrap.Wrapf(err, "couldn't remove directory %v", path)
	}
	log.Info("directory has been cleaned", "directory", path)
	return nil
}
