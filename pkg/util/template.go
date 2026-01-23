package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"text/template"

	ctrl "sigs.k8s.io/controller-runtime"

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

func CopyRpmPackageTemplates(
	ctx context.Context,
	templatesDest,
	assetsDir string,
	config *model.ConfigGoTemplating,
) error {
	l := ctrl.LoggerFrom(ctx)
	l.Info("Start handling RPM Package templates")

	// Define template paths
	makefileTemplatePath := path.Join(assetsDir, "templates/applications/rpm-package/Makefile.tmpl")
	rpmlintTemplatePath := path.Join(assetsDir, "templates/applications/rpm-package/.rpmlintrc.toml")

	specTemplatePath := path.Join(assetsDir, fmt.Sprintf("templates/applications/rpm-package/%s/spec.tmpl", config.Lang))
	serviceTemplatePath := path.Join(
		assetsDir,
		fmt.Sprintf("templates/applications/rpm-package/%s/service.tmpl", config.Lang),
	)

	if _, err := os.Stat(specTemplatePath); os.IsNotExist(err) {
		specTemplatePath = path.Join(assetsDir, "templates/applications/rpm-package/default/spec.tmpl")
		serviceTemplatePath = path.Join(assetsDir, "templates/applications/rpm-package/default/service.tmpl")
	} else if err != nil {
		return fmt.Errorf("failed to check if %q exists: %w", specTemplatePath, err)
	}

	// Define destination paths
	makefileDestPath := path.Join(templatesDest, "Makefile")
	if _, err := os.Stat(makefileDestPath); err == nil {
		makefileDestPath = path.Join(templatesDest, "Makefile.kuberocketci")
	}

	specDestPath := path.Join(templatesDest, fmt.Sprintf("%s.spec", config.Name))
	serviceDestPath := path.Join(templatesDest, fmt.Sprintf("%s.service", config.Name))
	rpmlintDestPath := path.Join(templatesDest, ".rpmlintrc.toml")

	// Create and render templates
	if err := createAndRenderTemplate(makefileDestPath, makefileTemplatePath, "Makefile.tmpl", config); err != nil {
		return err
	}

	if err := createAndRenderTemplate(specDestPath, specTemplatePath, "spec.tmpl", config); err != nil {
		return err
	}

	if err := createAndRenderTemplate(serviceDestPath, serviceTemplatePath, "service.tmpl", config); err != nil {
		return err
	}

	if err := CopyFile(rpmlintTemplatePath, rpmlintDestPath); err != nil {
		return err
	}

	l.Info("RPM Package templates have been copied and rendered")

	return nil
}

func createAndRenderTemplate(destPath, templatePath, templateName string, config *model.ConfigGoTemplating) error {
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", destPath, err)
	}

	defer func() {
		if cerr := destFile.Close(); cerr != nil {
			log.Error(cerr, "failed to close destination file", "file", destPath)
		}
	}()

	return renderTemplate(destFile, templatePath, templateName, config)
}

func CopyTemplate(
	ctx context.Context,
	deploymentScript,
	workDir,
	assetsDir string,
	cf *model.ConfigGoTemplating,
) error {
	switch deploymentScript {
	case HelmChartDeploymentScriptType:
		templatesDest := path.Join(workDir, "deploy-templates")

		if DoesDirectoryExist(templatesDest) {
			ctrl.LoggerFrom(ctx).Info("Deploy-templates folder already exists")
			return nil
		}

		return CopyHelmChartTemplates(deploymentScript, templatesDest, assetsDir, cf)

	case RpmPackageDeploymentScriptType:
		return CopyRpmPackageTemplates(ctx, workDir, assetsDir, cf)
	default:
		return errors.New("unsupported deployment type")
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
