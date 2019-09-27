package util

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	errWrap "github.com/pkg/errors"
	"io/ioutil"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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

func CopyTemplate(templatePath string, templateName string, config gerrit.GerritConfigGoTemplating) error {
	templatesDest := fmt.Sprintf("%v/%v/deploy-templates/%v.yaml", fmt.Sprintf("%v/oc-templates",
		config.CodebaseSettings.WorkDir), config.CodebaseSettings.Name,
		config.CodebaseSettings.Name)

	f, err := os.Create(templatesDest)
	if err != nil {
		return err
	}

	log.Info("Start rendering openshift templates", "path", templatePath)

	tmpl, err := template.New(templateName).ParseFiles(templatePath)
	if err != nil {
		return errWrap.Wrap(err, "unable to parse codebase deploy template")
	}

	err = tmpl.Execute(f, config)
	if err != nil {
		return errWrap.Wrap(err, "unable to render codebase deploy template")
	}

	log.Info("Openshift template has been rendered", "codebase", config.CodebaseSettings.Name)

	return nil
}
