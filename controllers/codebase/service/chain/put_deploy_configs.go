package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/template"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutDeployConfigs struct {
	client client.Client
	cr     repository.CodebaseRepository
	git    git.Git
}

func NewPutDeployConfigs(c client.Client, cr repository.CodebaseRepository, g git.Git) *PutDeployConfigs {
	return &PutDeployConfigs{client: c, cr: cr, git: g}
}

func (h *PutDeployConfigs) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	if c.Spec.DisablePutDeployTemplates {
		rLog.Info("skip of putting deploy templates to codebase due to specified flag")
		return nil
	}

	rLog.Info("Start pushing configs...")

	if err := h.tryToPushConfigs(ctx, c); err != nil {
		setFailedFields(c, codebaseApi.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs for %v codebase", c.Name)
	}

	rLog.Info("end pushing configs")

	return nil
}

func (h *PutDeployConfigs) tryToPushConfigs(ctx context.Context, c *codebaseApi.Codebase) error {
	edpN, err := helper.GetEDPName(ctx, h.client, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get edp name")
	}

	ps, err := h.cr.SelectProjectStatusValue(ctx, c.Name, *edpN)
	if err != nil {
		return errors.Wrapf(err, "couldn't get project_status value for %v codebase", c.Name)
	}

	var status = []string{util.ProjectTemplatesPushedStatus, util.ProjectVersionGoFilePushedStatus}
	if util.ContainsString(status, ps) {
		log.Info("skip pushing templates to gerrit. templates already pushed", "name", c.Name)

		return nil
	}

	s, err := util.GetSecret(h.client, "gerrit-project-creator", c.Namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	u := "project-creator"
	url := fmt.Sprintf("ssh://gerrit.%v:%v", c.Namespace, c.Name)
	wd := util.GetWorkDir(c.Name, c.Namespace)
	ad := util.GetAssetsDir()

	sshPort, err := util.GetGerritPort(h.client, c.Namespace)
	if err != nil {
		setFailedFields(c, codebaseApi.SetupDeploymentTemplates, err.Error())
		return errors.Wrap(err, "unable get gerrit port")
	}

	if !util.DoesDirectoryExist(wd) || util.IsDirectoryEmpty(wd) {
		err = h.cloneProjectRepoFromGerrit(*sshPort, idrsa, url, wd, ad)
		if err != nil {
			return err
		}
	}

	ru, err := util.GetRepoUrl(c)
	if err != nil {
		return errors.Wrap(err, "couldn't build repo url")
	}

	err = CheckoutBranch(ru, wd, c.Spec.DefaultBranch, h.git, c, h.client)
	if err != nil {
		return errors.Wrapf(err, "checkout default branch %v in Gerrit put_deploy_config has been failed", c.Spec.DefaultBranch)
	}

	err = template.PrepareTemplates(h.client, c, wd, ad)
	if err != nil {
		return fmt.Errorf("failed to prepare template: %w", err)
	}

	err = h.git.CommitChanges(wd, fmt.Sprintf("Add deployment templates for %v", c.Name))
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	err = h.git.PushChanges(idrsa, u, wd, *sshPort, "--all")
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	err = h.cr.UpdateProjectStatusValue(ctx, util.ProjectTemplatesPushedStatus, c.Name, *edpN)
	if err != nil {
		return errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectTemplatesPushedStatus, c.Name)
	}

	return nil
}

func (h *PutDeployConfigs) cloneProjectRepoFromGerrit(sshPort int32, idrsa, cloneSshUrl, wd, ad string) error {
	log.Info("start cloning repository from Gerrit", "ssh url", cloneSshUrl)

	err := h.git.CloneRepositoryBySsh(idrsa, "project-creator", cloneSshUrl, wd, sshPort)
	if err != nil {
		return fmt.Errorf("failed to clone git repository: %w", err)
	}

	destinationPath := fmt.Sprintf("%v/.git/hooks", wd)

	if err := util.CreateDirectory(destinationPath); err != nil {
		return errors.Wrapf(err, "couldn't create folder %v", destinationPath)
	}

	fileName := "commit-msg"
	src := fmt.Sprintf("%v/configs/%v", ad, fileName)
	dest := fmt.Sprintf("%v/%v", destinationPath, fileName)

	if err := util.CopyFile(src, dest); err != nil {
		return errors.Wrapf(err, "couldn't copy file %v", fileName)
	}

	return nil
}
