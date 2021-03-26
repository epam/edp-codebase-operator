package util

import (
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
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
		if _, err := os.Stat(fmt.Sprintf("%v/%v", dest, f.Name())); err == nil {
			log.Info("pipeline file already exists", "fileName", f.Name())
			continue
		}

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

func CopyHelmChartTemplates(deploymentScript, workDir string, config model.ConfigGoTemplating) error {
	log.Info("start handling Helm Chart templates", "codebase name", config.Name)
	templatesDest := fmt.Sprintf("%v/%v/%v/deploy-templates", workDir, "templates", config.Name)
	if DoesDirectoryExist(templatesDest) {
		log.Info("deploy-templates folder already exists")
		return nil
	}

	templateBasePath := fmt.Sprintf("/usr/local/bin/templates/applications/%v/%v", deploymentScript, config.PlatformType)

	log.Info("Paths", "templatesDest", templatesDest, "templateBasePath", templateBasePath)

	err := CreateDirectory(templatesDest)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", templateBasePath)

	valuesFileName := fmt.Sprintf("%v/%v.yaml", templatesDest, "values")
	valuesFile, err := os.Create(valuesFileName)
	if err != nil {
		return err
	}
	log.Info("file is created", "file", valuesFileName)

	chartFileName := fmt.Sprintf("%v/%v.yaml", templatesDest, "Chart")
	chartFile, err := os.Create(chartFileName)
	if err != nil {
		return err
	}
	log.Info("file is created", "file", chartFileName)

	templateFolder := fmt.Sprintf("%v/%v", templatesDest, TemplateFolder)
	err = CreateDirectory(templateFolder)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", templateFolder)

	testFolder := fmt.Sprintf("%v/%v/%v", templatesDest, TemplateFolder, TestFolder)
	err = CreateDirectory(testFolder)
	if err != nil {
		return err
	}
	log.Info("directory is created", "path", testFolder)

	templateSourceFolder := fmt.Sprintf("%v/%v", templateBasePath, TemplateFolder)
	err = CopyFiles(templateSourceFolder, templateFolder)
	if err != nil {
		return err
	}
	log.Info("files were copied", "from", templateSourceFolder, "to", templateFolder)

	testsSourceFolder := fmt.Sprintf("%v/%v/%v", templateBasePath, TemplateFolder, TestFolder)
	err = CopyFiles(testsSourceFolder, testFolder)
	if err != nil {
		return err
	}
	log.Info("files were copied", "from", testsSourceFolder, "to", testFolder)

	helmIgnoreSource := fmt.Sprintf("%v/%v", templateBasePath, HelmIgnoreFile)
	helmIgnoreFile := fmt.Sprintf("%v/%v", templatesDest, HelmIgnoreFile)
	err = CopyFile(helmIgnoreSource, helmIgnoreFile)
	if err != nil {
		return err
	}
	log.Info("file were copied", "from", helmIgnoreFile, "to", templatesDest)

	if err := renderTemplate(valuesFile, fmt.Sprintf("%v/%v", templateBasePath, ChartValuesTemplate), ChartValuesTemplate, config); err != nil {
		return err
	}

	if err := renderTemplate(chartFile, fmt.Sprintf("%v/%v", templateBasePath, ChartTemplate), ChartTemplate, config); err != nil {
		return err
	}

	templateFolderFilesList, err := GetListFilesInDirectory(fmt.Sprintf("%v/%v", templatesDest, TemplateFolder))
	for _, file := range templateFolderFilesList {
		if file.IsDir() {
			continue
		}
		if err := ReplaceStringInFile(fmt.Sprintf("%v/%v/%v", templatesDest, TemplateFolder, file.Name()),"REPLACE_IT", config.Name); err != nil {
			return err
		}
	}

	if err := ReplaceStringInFile(fmt.Sprintf("%v/%v/%v/%v", templatesDest, TemplateFolder, TestFolder, TestFile),"REPLACE_IT", config.Name); err != nil {
		return err
	}

	log.Info("end handling Helm Chart templates", "codebase name", config.Name)
	return nil
}

func CopyOpenshiftTemplate(deploymentScript, workDir string, config model.ConfigGoTemplating) error {
	log.Info("start handling Openshift template", "codebase name", config.Name)
	templatesDest := fmt.Sprintf("%v/%v/%v/deploy-templates", workDir, "templates", config.Name)
	if DoesDirectoryExist(templatesDest) {
		log.Info("deploy-templates folder already exists")
		return nil
	}

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

func CopyTemplate(deploymentScript, workDir string, cf model.ConfigGoTemplating) error {
	if deploymentScript == HelmChartDeploymentScriptType {
		return CopyHelmChartTemplates(deploymentScript, workDir, cf)
	}
	return CopyOpenshiftTemplate(deploymentScript, workDir, cf)
}

func renderTemplate(file *os.File, templateBasePath, templateName string, config model.ConfigGoTemplating) error {
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
