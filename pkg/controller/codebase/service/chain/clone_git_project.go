package chain

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type CloneGitProject struct {
	client client.Client
	git    git.Git
}

const (
	repoNotReady = "NOT_READY"
)

func NewCloneGitProject(client client.Client, git git.Git) *CloneGitProject {
	return &CloneGitProject{client: client, git: git}
}

func (h *CloneGitProject) ServeRequest(_ context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("Start cloning project...")
	if c.Spec.GitUrlPath != nil && *c.Spec.GitUrlPath == repoNotReady {
		rLog.Info("postpone reconciliation, repo is not ready")
		return PostponeError{Timeout: time.Second * 30}
	}

	rLog.Info("codebase data", "spec", c.Spec)
	if err := h.setIntermediateSuccessFields(c, codebaseApi.AcceptCodebaseRegistration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	log.Info("Setting path for local Git folder", "path", wd)
	if err := util.CreateDirectory(wd); err != nil {
		setFailedFields(c, codebaseApi.ImportProject, err.Error())
		return err
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.ImportProject, err.Error())
		return errors.Wrapf(err, "an error has occurred while getting %v GitServer", c.Spec.GitServer)
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.ImportProject, err.Error())
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	ru := fmt.Sprintf("ssh://%v:%d%v", gs.GitHost, gs.SshPort, *c.Spec.GitUrlPath)

	if !util.DoesDirectoryExist(wd) || util.IsDirectoryEmpty(wd) {
		if err := h.git.CloneRepositoryBySsh(k, u, ru, wd, gs.SshPort); err != nil {
			setFailedFields(c, codebaseApi.ImportProject, err.Error())
			return errors.Wrapf(err, "an error has occurred while cloning repository %v", ru)
		}
	}
	rLog.Info("end cloning project")
	return nil
}

func (h *CloneGitProject) setIntermediateSuccessFields(c *codebaseApi.Codebase, action codebaseApi.ActionType) error {
	c.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Action:          action,
		Result:          codebaseApi.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    c.Status.FailureCount,
		Git:             c.Status.Git,
	}

	if err := h.client.Status().Update(context.TODO(), c); err != nil {
		if err := h.client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
