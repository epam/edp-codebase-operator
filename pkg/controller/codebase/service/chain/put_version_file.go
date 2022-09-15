package chain

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
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

func NewPutVersionFile(client client.Client, cr repository.CodebaseRepository, git git.Git) *PutVersionFile {
	return &PutVersionFile{client: client, cr: cr, git: git}
}

func (h *PutVersionFile) ServeRequest(_ context.Context, c *codebaseApi.Codebase) error {
	if strings.ToLower(c.Spec.Lang) != goLang ||
		(strings.ToLower(c.Spec.Lang) == goLang && c.Spec.Versioning.Type == "edp") {
		return nil
	}

	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("start putting VERSION file...")

	name, err := helper.GetEDPName(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return err
	}

	exists, err := h.versionFileExists(c.Name, *name)
	if err != nil {
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return err
	}

	if exists {
		log.Info("skip pushing VERSION file to Git provider. file already exists",
			"name", c.Name)
		return nil
	}

	if err := h.tryToPutVersionFile(c, util.GetWorkDir(c.Name, c.Namespace)); err != nil {
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectVersionGoFilePushedStatus, c.Name, *name); err != nil {
		err := errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectVersionGoFilePushedStatus, c.Name)
		setFailedFields(c, codebaseApi.PutVersionFile, err.Error())
		return err
	}

	rLog.Info("end putting VERSION file...")
	return nil
}

func (h *PutVersionFile) versionFileExists(codebaseName, edpName string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(codebaseName, edpName)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if *ps == util.ProjectVersionGoFilePushedStatus {
		return true, nil
	}

	return false, nil
}

func (h *PutVersionFile) tryToPutVersionFile(c *codebaseApi.Codebase, projectPath string) error {
	path := fmt.Sprintf("%v/%v", projectPath, versionFileName)
	if err := createFile(path); err != nil {
		return errors.Wrapf(err, "couldn't create file %v", path)
	}

	if err := writeFile(path); err != nil {
		return errors.Wrapf(err, "couldn't write to file %v", path)
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	if err := CheckoutBranch(ru, projectPath, c.Spec.DefaultBranch, h.git, c, h.client); err != nil {
		return errors.Wrapf(err, "checkout default branch %v in Gerrit has been failed", c.Spec.DefaultBranch)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := h.pushChanges(projectPath, k, u); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing %v for %v codebase", versionFileName, c.Name)
	}

	return nil
}

func (h *PutVersionFile) pushChanges(projectPath, privateKey, user string) error {
	if err := h.git.CommitChanges(projectPath, fmt.Sprintf("Add %v file", versionFileName)); err != nil {
		return err
	}

	if err := h.git.PushChanges(privateKey, user, projectPath, "--all"); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v project", projectPath)
	}

	return nil
}

func createFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsExist(err) {
		log.Info("File already exists. skip creating.", "name", filePath)
		return nil
	}

	var file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	log.Info("File has been created.", "name", filePath)
	return nil
}

func writeFile(filePath string) error {
	var file, err = os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.WriteString(initVersion); err != nil {
		return err
	}

	if err = file.Sync(); err != nil {
		return err
	}

	log.Info("File has been updated.", "name", filePath)
	return nil
}
