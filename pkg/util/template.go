package util

import (
	"errors"
	"fmt"
	"os"
	"path"
	"text/template"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
)

const (
	logDirCreatedMessage = "directory is created"
	logPathKey           = "path"
	fileLogKey           = "file"
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

	log.Info("file is created", fileLogKey, valuesFileName)

	chartFileName := path.Join(templatesDest, "Chart.yaml")

	chartFile, err := os.Create(chartFileName)
	if err != nil {
		return fmt.Errorf("failed to create chart file %q: %w", chartFileName, err)
	}

	log.Info("file is created", fileLogKey, chartFileName)

	readmeFileName := path.Join(templatesDest, "README.md")

	readmeFile, err := os.Create(readmeFileName)
	if err != nil {
		return fmt.Errorf("failed to create chart file %q: %w", readmeFileName, err)
	}

	log.Info("file is created", fileLogKey, readmeFileName)

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

	readmegoFileName := path.Join(templateBasePath, "README.md.gotmpl")

	bytesRead, err := os.ReadFile(readmegoFileName)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", readmegoFileName, err)
	}

	readmeFileNameCopy := path.Join(templatesDest, "README.md.gotmpl")

	if err = os.WriteFile(readmeFileNameCopy, bytesRead, readWriteMode); err != nil {
		return fmt.Errorf("failed to write file %q: %w", readmeFileNameCopy, err)
	}

	log.Info("file is copied", fileLogKey, readmeFileNameCopy)

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

	err = renderTemplate(readmeFile, path.Join(templateBasePath, ReadmeTemplate), ReadmeTemplate, config)
	if err != nil {
		return err
	}

	templateFolderFilesList, err := GetListFilesInDirectory(path.Join(templatesDest, TemplateFolder))
	if err != nil {
		return fmt.Errorf("failed to GetListFilesInDirectory: %w", err)
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

func CopyRpmPackageTemplates(templatesDest, assetsDir string, config *model.ConfigGoTemplating) error {
	log.Info("start handling RPM Package templates", logCodebaseNameKey, config.Name)

	// Define template paths
	makefileTemplatePath := path.Join(assetsDir, "templates/applications/rpm-package/Makefile.tmpl")
	rpmlintTemplatePath := path.Join(assetsDir, "templates/applications/rpm-package/.rpmlintrc.toml")
	specTemplatePath := path.Join(assetsDir, fmt.Sprintf("templates/applications/rpm-package/%s/spec.tmpl", config.Lang))
	serviceTemplatePath := path.Join(assetsDir, fmt.Sprintf("templates/applications/rpm-package/%s/service.tmpl", config.Lang))
	// Define destination paths
	makefileDestPath := path.Join(templatesDest, "Makefile")
	specDestPath := path.Join(templatesDest, fmt.Sprintf("%s.spec", config.Name))
	serviceDestPath := path.Join(templatesDest, fmt.Sprintf("%s.service", config.Name))
	rpmlintDestPath := path.Join(templatesDest, ".rpmlintrc.toml")

	// Create destination files
	makefileDestFile, err := os.Create(makefileDestPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", makefileDestPath, err)
	}

	specDestFile, err := os.Create(specDestPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", specDestPath, err)
	}

	serviceDestFile, err := os.Create(serviceDestPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", serviceDestPath, err)
	}

	// Render and copy templates
	err = renderTemplate(makefileDestFile, makefileTemplatePath, "Makefile.tmpl", config)
	if err != nil {
		return err
	}

	err = renderTemplate(specDestFile, specTemplatePath, "spec.tmpl", config)
	if err != nil {
		return err
	}

	err = renderTemplate(serviceDestFile, serviceTemplatePath, "service.tmpl", config)
	if err != nil {
		return err
	}

	err = CopyFile(rpmlintTemplatePath, rpmlintDestPath)
	if err != nil {
		return err
	}

	log.Info("RPM Package templates have been copied and rendered", logCodebaseNameKey, config.Name)

	return nil
}

func CopyTemplate(deploymentScript, workDir, assetsDir string, cf *model.ConfigGoTemplating) error {
	switch deploymentScript {
	case HelmChartDeploymentScriptType:
		templatesDest := path.Join(workDir, "deploy-templates")

		if DoesDirectoryExist(templatesDest) {
			log.Info("deploy-templates folder already exists")
			return nil
		}

		return CopyHelmChartTemplates(deploymentScript, templatesDest, assetsDir, cf)

	case RpmPackageDeploymentScriptType:
		return CopyRpmPackageTemplates(workDir, assetsDir, cf)
	default:
		return errors.New("Unsupported deployment type")
	}
}

func renderTemplate(file *os.File, templateBasePath, templateName string, config *model.ConfigGoTemplating) error {
	log.Info("start rendering deploy template", logPathKey, templateBasePath)

	tmpl, err := template.New(templateName).ParseFiles(templateBasePath)
	if err != nil {
		return fmt.Errorf("failed to parse codebase deploy template: %w", err)
	}

	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("failed to render codebase deploy template: %w", err)
	}

	log.Info("template has been rendered", "codebase", config.Name)

	return nil
}
