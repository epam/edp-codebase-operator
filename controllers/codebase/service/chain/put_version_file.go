package chain

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutVersionFile struct {
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

const (
	versionFileName = "VERSION"
	initVersion     = "0.0.1"
	goLang          = "go"
)

func NewPutVersionFile(c client.Client, cr repository.CodebaseRepository, g git.Git) *PutVersionFile {
	return &PutVersionFile{client: c, cr: cr, git: g}
}

func (h *PutVersionFile) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	if !strings.EqualFold(c.Spec.Lang, goLang) ||
		(strings.EqualFold(c.Spec.Lang, goLang) && c.Spec.Versioning.Type == "edp") {
		return nil
	}

	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("start putting VERSION file...")

	name, err := helper.GetEDPName(ctx, h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return fmt.Errorf("failed to get EDP name: %w", err)
	}

	exists, err := h.versionFileExists(ctx, c.Name, *name)
	if err != nil {
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return err
	}

	if exists {
		log.Info("skip pushing VERSION file to Git provider. file already exists",
			"name", c.Name)
		return nil
	}

	err = h.tryToPutVersionFile(c, util.GetWorkDir(c.Name, c.Namespace))
	if err != nil {
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return err
	}

	err = h.cr.UpdateProjectStatusValue(ctx, util.ProjectVersionGoFilePushedStatus, c.Name, *name)
	if err != nil {
		err = fmt.Errorf("failed to set project_status - %v value, codebase - %v: %w",
			util.ProjectVersionGoFilePushedStatus, c.Name, err)
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())

		return err
	}

	rLog.Info("end putting VERSION file...")

	return nil
}

func (h *PutVersionFile) versionFileExists(ctx context.Context, codebaseName, edpName string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(ctx, codebaseName, edpName)
	if err != nil {
		return false, fmt.Errorf("failed to get project_status value for %v codebase: %w", codebaseName, err)
	}

	if ps == util.ProjectVersionGoFilePushedStatus {
		return true, nil
	}

	return false, nil
}

func (h *PutVersionFile) tryToPutVersionFile(c *codebaseApi.Codebase, projectPath string) error {
	path := fmt.Sprintf("%v/%v", projectPath, versionFileName)
	if err := createFile(path); err != nil {
		return fmt.Errorf("failed to create file %v: %w", path, err)
	}

	if err := writeFile(path); err != nil {
		return fmt.Errorf("failed to write to file %v: %w", path, err)
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get GitServer: %w", err)
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get %v secret: %w", gs.NameSshKeySecret, err)
	}

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	if err := CheckoutBranch(ru, projectPath, c.Spec.DefaultBranch, h.git, c, h.client); err != nil {
		return fmt.Errorf("failed to checkout default branch %v in Gerrit: %w", c.Spec.DefaultBranch, err)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	p := gs.SshPort

	if err := h.pushChanges(projectPath, k, u, p); err != nil {
		return fmt.Errorf("failed to push %v for %v codebase: %w", versionFileName, c.Name, err)
	}

	return nil
}

func (h *PutVersionFile) pushChanges(projectPath, privateKey, user string, port int32) error {
	if err := h.git.CommitChanges(projectPath, fmt.Sprintf("Add %v file", versionFileName)); err != nil {
		return fmt.Errorf("failed to commit changes to Git server: %w", err)
	}

	if err := h.git.PushChanges(privateKey, user, projectPath, port, "--all"); err != nil {
		return fmt.Errorf("failed to push changes for %v project: %w", projectPath, err)
	}

	return nil
}

func createFile(filePath string) (err error) {
	_, err = os.Stat(filePath)
	if errors.Is(err, fs.ErrExist) {
		log.Info("File already exists. skip creating.", "filePath", filePath)
		return nil
	}

	// ignore all other errors
	err = nil

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", filePath, err)
	}

	defer util.CloseWithErrorCapture(&err, file, "failed to close file: %s", filePath)

	log.Info("File has been created.", "filePath", filePath)

	return
}

func writeFile(filePath string) (err error) {
	const readWritePermBits = 0o644

	file, err := os.OpenFile(filePath, os.O_RDWR, readWritePermBits)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filePath, err)
	}

	defer util.CloseWithErrorCapture(&err, file, "failed to close file: %s", filePath)

	_, err = file.WriteString(initVersion)
	if err != nil {
		return fmt.Errorf("failed to writeS file %q: %w", filePath, err)
	}

	err = file.Sync()
	if err != nil {
		return fmt.Errorf("failed to commit file %q: %w", filePath, err)
	}

	log.Info("File has been updated.", "filePath", filePath)

	return
}
