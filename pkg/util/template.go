package util

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/pkg/errors"
)

func CopyHelmChartTemplates(deploymentScript, templatesDest, assetsDir string, config model.ConfigGoTemplating) error {
	log.Info("start handling Helm Chart templates", "codebase_name", config.Name)

	templateBasePath := fmt.Sprintf("%v/templates/applications/%v/%v", assetsDir, deploymentScript, config.PlatformType)

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
	if err != nil {
		return errors.Wrapf(err, "Unable to GetListFilesInDirectory")
	}

	for _, file := range templateFolderFilesList {
		if file.IsDir() {
			continue
		}
		if err := ReplaceStringInFile(fmt.Sprintf("%v/%v/%v", templatesDest, TemplateFolder, file.Name()), "REPLACE_IT", config.Name); err != nil {
			return err
		}
	}

	if err := ReplaceStringInFile(fmt.Sprintf("%v/%v/%v/%v", templatesDest, TemplateFolder, TestFolder, TestFile), "REPLACE_IT", config.Name); err != nil {
		return err
	}

	log.Info("end handling Helm Chart templates", "codebase_name", config.Name)
	return nil
}

func CopyOpenshiftTemplate(deploymentScript, templatesDest, assetsDir string, config model.ConfigGoTemplating) error {
	log.Info("start handling Openshift template", "codebase_name", config.Name)

	templateBasePath := fmt.Sprintf("%v/templates/applications/%v/%v", assetsDir, deploymentScript, strings.ToLower(config.Lang))
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
	log.Info("end handling Openshift template", "codebase_name", config.Name)
	return nil
}

func CopyTemplate(deploymentScript, workDir, assetsDir string, cf model.ConfigGoTemplating) error {
	templatesDest := fmt.Sprintf("%v/deploy-templates", workDir)
	if DoesDirectoryExist(templatesDest) {
		log.Info("deploy-templates folder already exists")
		return nil
	}

	switch deploymentScript {
	case HelmChartDeploymentScriptType:
		return CopyHelmChartTemplates(deploymentScript, templatesDest, assetsDir, cf)
	case "openshift-template":
		return CopyOpenshiftTemplate(deploymentScript, templatesDest, assetsDir, cf)
	default:
		return errors.New("Unsupported deployment type")
	}
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
