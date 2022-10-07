package chain

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutGitlabCiFile struct {
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func NewPutGitlabCiFile(c client.Client, cr repository.CodebaseRepository, g git.Git) *PutGitlabCiFile {
	return &PutGitlabCiFile{client: c, cr: cr, git: g}
}

func (h *PutGitlabCiFile) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("start creating gitlab ci file...")

	name, err := helper.GetEDPName(ctx, h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.PutGitlabCIFile, err.Error())
		return err
	}

	exists, err := h.gitlabCiFileExists(ctx, c.Name, *name)
	if err != nil {
		setFailedFields(c, codebaseApi.PutGitlabCIFile, err.Error())
		return err
	}

	if exists {
		log.Info("skip pushing gitlab ci file to Git provider. file already exists", "name", c.Name)
		return nil
	}

	if err := h.tryToPutGitlabCIFile(c); err != nil {
		setFailedFields(c, codebaseApi.PutGitlabCIFile, err.Error())
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(ctx, util.GitlabCi, c.Name, *name); err != nil {
		err = errors.Wrapf(err, "couldn't set project_status %v value for %v codebase", util.GitlabCi, c.Name)
		setFailedFields(c, codebaseApi.PutGitlabCIFile, err.Error())
		return err
	}

	rLog.Info("end creating gitlab ci file...")
	return nil
}

func (h *PutGitlabCiFile) tryToPutGitlabCIFile(c *codebaseApi.Codebase) error {
	if err := h.parseTemplate(c); err != nil {
		return err
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	p := gs.SshPort
	if err := h.pushChanges(util.GetWorkDir(c.Name, c.Namespace), k, u, c.Spec.DefaultBranch, p); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing %v for %v codebase", versionFileName, c.Name)
	}
	return nil
}

func (h *PutGitlabCiFile) pushChanges(projectPath, privateKey, user, defaultBranch string, port int32) error {
	if err := h.git.CommitChanges(projectPath, fmt.Sprintf("Add %v file", util.GitlabCi)); err != nil {
		return err
	}

	if err := h.git.PushChanges(privateKey, user, projectPath, port, defaultBranch); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v project", projectPath)
	}

	return nil
}

func (h *PutGitlabCiFile) parseTemplate(c *codebaseApi.Codebase) error {
	tp := fmt.Sprintf("%v/templates/gitlabci/%v/%v-%v.tmpl",
		util.GetAssetsDir(),
		platform.GetPlatformType(), strings.ToLower(*c.Spec.Framework), strings.ToLower(c.Spec.BuildTool))

	wd := util.GetWorkDir(c.Name, c.Namespace)
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

	return parseTemplate(tp, gitlabCiFile, data)
}

func (h *PutGitlabCiFile) gitlabCiFileExists(ctx context.Context, codebaseName, edpName string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(ctx, codebaseName, edpName)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if util.ContainsString([]string{util.GitlabCiFilePushedStatus, util.ProjectVersionGoFilePushedStatus}, ps) {
		return true, nil
	}

	return false, nil
}

func parseTemplate(templatePath, gitlabCiFile string, data interface{}) (err error) {
	f, err := os.Create(gitlabCiFile)
	if err != nil {
		return err
	}

	defer util.CloseWithErrorCapture(&err, f, "failed to close gitlab CI file")

	log.Info("file has been created.", "name", gitlabCiFile)

	split := strings.Split(templatePath, "/")
	tmpl, err := template.New(split[len(split)-1]).ParseFiles(templatePath)
	if err != nil {
		return err
	}

	err = tmpl.Execute(f, data)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse template %v", templatePath)
	}

	log.Info("template has been rendered", "path", gitlabCiFile)

	return
}

func getEdpComponentName() string {
	if platform.IsK8S() {
		return platform.K8S
	}
	return platform.Openshift
}
