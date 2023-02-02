package chain

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type CloneGitProject struct {
	client client.Client
	git    git.Git
}

const (
	repoNotReady = "NOT_READY"
)

func NewCloneGitProject(c client.Client, g git.Git) *CloneGitProject {
	return &CloneGitProject{client: c, git: g}
}

func (h *CloneGitProject) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	const defaultPostponeTime = 30 * time.Second

	rLog := log.WithValues("codebase_name", c.Name)

	rLog.Info("Start cloning project...")

	if c.Spec.GitUrlPath != nil && *c.Spec.GitUrlPath == repoNotReady {
		rLog.Info("postpone reconciliation, repo is not ready")
		return PostponeError{Timeout: defaultPostponeTime}
	}

	rLog.Info("codebase data", "spec", c.Spec)

	if err := setIntermediateSuccessFields(ctx, h.client, c, codebaseApi.AcceptCodebaseRegistration); err != nil {
		return fmt.Errorf("failed to update Codebase status %v: %w", c.Name, err)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)

	log.Info("Setting path for local Git folder", "path", wd)

	if err := util.CreateDirectory(wd); err != nil {
		setFailedFields(c, codebaseApi.ImportProject, err.Error())

		return fmt.Errorf("failed to create directory %q: %w", wd, err)
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.ImportProject, err.Error())
		return fmt.Errorf("failed to get Gitserver %v: %w", c.Spec.GitServer, err)
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.ImportProject, err.Error())
		return fmt.Errorf("failed to get secret %v: %w", gs.NameSshKeySecret, err)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	ru := fmt.Sprintf("ssh://%v:%d%v", gs.GitHost, gs.SshPort, *c.Spec.GitUrlPath)

	if !util.DoesDirectoryExist(wd) || util.IsDirectoryEmpty(wd) {
		if err := h.git.CloneRepositoryBySsh(k, u, ru, wd, gs.SshPort); err != nil {
			setFailedFields(c, codebaseApi.ImportProject, err.Error())
			return fmt.Errorf("failed to clone repository %v: %w", ru, err)
		}
	}

	rLog.Info("end cloning project")

	return nil
}
