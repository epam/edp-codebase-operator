package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/platform"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
)

type PutGitlabCiFile struct {
	next   handler.CodebaseHandler
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func (h PutGitlabCiFile) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("start creating gitlab ci file...")

	name, err := helper.GetEDPName(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, v1alpha1.PutGitlabCIFile, err.Error())
		return err
	}

	exists, err := h.gitlabCiFileExists(c.Name, *name)
	if err != nil {
		setFailedFields(c, v1alpha1.PutGitlabCIFile, err.Error())
		return err
	}

	if exists {
		log.Info("skip pushing gitlab ci file to Git provider. file already exists", "name", c.Name)
		return nextServeOrNil(h.next, c)
	}

	if err := h.tryToPutGitlabCIFile(c); err != nil {
		setFailedFields(c, v1alpha1.PutGitlabCIFile, err.Error())
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.GitlabCi, c.Name, *name); err != nil {
		err = errors.Wrapf(err, "couldn't set project_status %v value for %v codebase", util.GitlabCi, c.Name)
		setFailedFields(c, v1alpha1.PutGitlabCIFile, err.Error())
		return err
	}

	rLog.Info("end creating gitlab ci file...")
	return nextServeOrNil(h.next, c)
}

func (h PutGitlabCiFile) tryToPutGitlabCIFile(c *v1alpha1.Codebase) error {
	if err := h.parseTemplate(c); err != nil {
		return err
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := getSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := h.pushChanges(util.GetWorkDir(c.Name, c.Namespace), k, u); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing %v for %v codebase", versionFileName, c.Name)
	}
	return nil
}

func (h PutGitlabCiFile) pushChanges(projectPath, privateKey, user string) error {
	if err := h.git.CommitChanges(projectPath, fmt.Sprintf("Add %v file", util.GitlabCi)); err != nil {
		return err
	}

	if err := h.git.PushChanges(privateKey, user, projectPath); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v project", projectPath)
	}

	return nil
}

func (h PutGitlabCiFile) parseTemplate(c *v1alpha1.Codebase) error {
	tp := getTemplatePath(*c.Spec.Framework, c.Spec.BuildTool)

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v/%v", c.Namespace, c.Name, "templates", c.Name)
	gitlabCiFile := fmt.Sprintf("%v/%v", wd, ".gitlab-ci.yml")

	component, err := util.GetEdpComponent(h.client, getEdpComponentName(), c.Namespace)
	if err != nil {
		return err
	}

	data := struct {
		CodebaseName   string
		Namespace      string
		VersioningType string
		ClusterUrl     string
	}{
		c.Name,
		c.Namespace,
		string(c.Spec.Versioning.Type),
		component.Spec.Url,
	}
	if err := parseTemplate(tp, gitlabCiFile, data); err != nil {
		return err
	}
	return nil
}

func getEdpComponentName() string {
	if platform.IsK8S() {
		return util.KubernetesConsoleEdpComponent
	}
	return platform.Openshift
}

func (h PutGitlabCiFile) gitlabCiFileExists(codebaseName, edpName string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(codebaseName, edpName)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if util.ContainsString([]string{util.GitlabCiFilePushedStatus, util.ProjectVersionGoFilePushedStatus}, *ps) {
		return true, nil
	}

	return false, nil
}

func getTemplatePath(framework, buildTool string) string {
	return fmt.Sprintf("/usr/local/bin/templates/gitlabci/%v/%v-%v.tmpl",
		platform.GetPlatformType(), strings.ToLower(framework), strings.ToLower(buildTool))
}

func parseTemplate(templatePath, gitlabCiFile string, data interface{}) error {
	var f, err = os.Create(gitlabCiFile)
	if err != nil {
		return err
	}
	defer f.Close()
	log.Info("file has been created.", "name", gitlabCiFile)

	split := strings.Split(templatePath, "/")
	tmpl, err := template.New(split[len(split)-1]).ParseFiles(templatePath)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, data); err != nil {
		return errors.Wrapf(err, "couldn't parse template %v", templatePath)
	}
	log.Info("template has been rendered", "path", gitlabCiFile)
	return nil
}
