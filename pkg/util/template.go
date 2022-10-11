package util

import (
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
)

const (
	logDirCreatedMessage = "directory is created"
	logPathKey           = "path"
)

func CopyHelmChartTemplates(deploymentScript, templatesDest, assetsDir string, config *model.ConfigGoTemplating) error {
	log.Info("start handling Helm Chart templates", logCodebaseNameKey, config.Name)

	templateBasePath := path.Join(assetsDir, "templates/applications", deploymentScript, config.PlatformType)

	log.Info("Paths", "templatesDest", templatesDest, "templateBasePath", templateBasePath)

	err := CreateDirectory(templatesDest)
	if err != nil {
		return err
	}

	log.Info(logDirCreatedMessage, logPathKey, templateBasePath)

	valuesFileName := path.Join(templatesDest, "values.yaml")

	valuesFile, err := os.Create(valuesFileName)
	if err != nil {
		return fmt.Errorf("failed to create Values file %q: %w", valuesFileName, err)
	}

	log.Info("file is created", "file", valuesFileName)

	chartFileName := path.Join(templatesDest, "Chart.yaml")

	chartFile, err := os.Create(chartFileName)
	if err != nil {
		return fmt.Errorf("failed to create chart file %q: %w", chartFileName, err)
	}

	log.Info("file is created", "file", chartFileName)

	templateFolder := path.Join(templatesDest, TemplateFolder)

	err = CreateDirectory(templateFolder)
	if err != nil {
		return err
	}

	log.Info(logDirCreatedMessage, logPathKey, templateFolder)

	testFolder := path.Join(templatesDest, TemplateFolder, TestFolder)

	err = CreateDirectory(testFolder)
	if err != nil {
		return err
	}

	log.Info(logDirCreatedMessage, logPathKey, testFolder)

	templateSourceFolder := path.Join(templateBasePath, TemplateFolder)

	err = CopyFiles(templateSourceFolder, templateFolder)
	if err != nil {
		return err
	}

	log.Info("files were copied", "from", templateSourceFolder, "to", templateFolder)

	testsSourceFolder := path.Join(templateBasePath, TemplateFolder, TestFolder)

	err = CopyFiles(testsSourceFolder, testFolder)
	if err != nil {
		return err
	}

	log.Info("files were copied", "from", testsSourceFolder, "to", testFolder)

	helmIgnoreSource := path.Join(templateBasePath, HelmIgnoreFile)
	helmIgnoreFile := path.Join(templatesDest, HelmIgnoreFile)

	err = CopyFile(helmIgnoreSource, helmIgnoreFile)
	if err != nil {
		return err
	}

	log.Info("file were copied", "from", helmIgnoreFile, "to", templatesDest)

	err = renderTemplate(valuesFile, path.Join(templateBasePath, ChartValuesTemplate), ChartValuesTemplate, config)
	if err != nil {
		return err
	}

	err = renderTemplate(chartFile, path.Join(templateBasePath, ChartTemplate), ChartTemplate, config)
	if err != nil {
		return err
	}

	templateFolderFilesList, err := GetListFilesInDirectory(path.Join(templatesDest, TemplateFolder))
	if err != nil {
		return errors.Wrapf(err, "Unable to GetListFilesInDirectory")
	}

	for _, file := range templateFolderFilesList {
		if file.IsDir() {
			continue
		}

		err = ReplaceStringInFile(path.Join(templatesDest, TemplateFolder, file.Name()), "REPLACE_IT", config.Name)
		if err != nil {
			return err
		}
	}

	err = ReplaceStringInFile(path.Join(templatesDest, TemplateFolder, TestFolder, TestFile), "REPLACE_IT", config.Name)
	if err != nil {
		return err
	}

	log.Info("end handling Helm Chart templates", logCodebaseNameKey, config.Name)

	return nil
}

func CopyOpenshiftTemplate(deploymentScript, templatesDest, assetsDir string, config *model.ConfigGoTemplating) error {
	log.Info("start handling Openshift template", logCodebaseNameKey, config.Name)

	templateBasePath := path.Join(assetsDir, "templates/applications", deploymentScript, strings.ToLower(config.Lang))
	templateName := fmt.Sprintf("%v.tmpl", strings.ToLower(config.Lang))

	log.Info("Paths", "templatesDest", templatesDest, "templateBasePath", templateBasePath,
		"templateName", templateName)

	err := CreateDirectory(templatesDest)
	if err != nil {
		return err
	}

	log.Info(logDirCreatedMessage, logPathKey, templatesDest)

	fp := path.Join(templatesDest, config.Name+".yaml")

	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf("failed to create openshift template file: %w", err)
	}

	log.Info("file is created", logPathKey, fp)

	err = renderTemplate(f, path.Join(templateBasePath, templateName), templateName, config)
	if err != nil {
		return err
	}

	log.Info("end handling Openshift template", logCodebaseNameKey, config.Name)

	return nil
}

func CopyTemplate(deploymentScript, workDir, assetsDir string, cf *model.ConfigGoTemplating) error {
	templatesDest := path.Join(workDir, "deploy-templates")

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

func renderTemplate(file *os.File, templateBasePath, templateName string, config *model.ConfigGoTemplating) error {
	log.Info("start rendering deploy template", logPathKey, templateBasePath)

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
