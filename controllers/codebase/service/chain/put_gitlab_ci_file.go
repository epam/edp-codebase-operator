package chain

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
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

		return fmt.Errorf("failed to fetch EDP Name: %w", err)
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
		err = fmt.Errorf("failed to set project_status %v value for %v codebase: %w", util.GitlabCi, c.Name, err)
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
		return fmt.Errorf("failed to fetch GitServer: %w", err)
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get %v secret: %w", gs.NameSshKeySecret, err)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	p := gs.SshPort

	if err := h.pushChanges(util.GetWorkDir(c.Name, c.Namespace), k, u, c.Spec.DefaultBranch, p); err != nil {
		return fmt.Errorf("failed to push %v for %v codebase: %w", util.GitlabCi, c.Name, err)
	}

	return nil
}

func (h *PutGitlabCiFile) pushChanges(projectPath, privateKey, user, defaultBranch string, port int32) error {
	if err := h.git.CommitChanges(projectPath, fmt.Sprintf("Add %v file", util.GitlabCi)); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := h.git.PushChanges(privateKey, user, projectPath, port, defaultBranch); err != nil {
		return fmt.Errorf("failed to push changes for %v project: %w", projectPath, err)
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
		return fmt.Errorf("failed to fetch EdpComponent: %w", err)
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
		return false, fmt.Errorf("failed to get project_status value for %v codebase: %w", codebaseName, err)
	}

	if util.ContainsString([]string{util.GitlabCiFilePushedStatus}, ps) {
		return true, nil
	}

	return false, nil
}

func parseTemplate(templatePath, gitlabCiFile string, data interface{}) (err error) {
	f, err := os.Create(gitlabCiFile)
	if err != nil {
		return fmt.Errorf("failed to create GitlabCI file %q: %w", gitlabCiFile, err)
	}

	defer util.CloseWithErrorCapture(&err, f, "failed to close gitlab CI file")

	log.Info("file has been created.", "name", gitlabCiFile)

	split := strings.Split(templatePath, "/")

	tmpl, err := template.New(split[len(split)-1]).ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	err = tmpl.Execute(f, data)
	if err != nil {
		return fmt.Errorf("failed to parse template %v: %w", templatePath, err)
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
