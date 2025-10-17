package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/template"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutDeployConfigs struct {
	client client.Client
	git    git.Git
}

func NewPutDeployConfigs(c client.Client, g git.Git) *PutDeployConfigs {
	return &PutDeployConfigs{client: c, git: g}
}

func (h *PutDeployConfigs) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if c.Spec.DisablePutDeployTemplates {
		log.Info("Skip of putting deploy templates to codebase due to specified flag")
		return nil
	}

	if c.Spec.Type != util.Application {
		log.Info("Skip putting deploy templates to codebase because it is not application")
		return nil
	}

	log.Info("Start pushing configs")

	if err := h.tryToPushConfigs(ctx, c); err != nil {
		setFailedFields(c, codebaseApi.SetupDeploymentTemplates, err.Error())
		return fmt.Errorf("failed to push deploy configs for %v codebase: %w", c.Name, err)
	}

	log.Info("End pushing configs")

	return nil
}

func (h *PutDeployConfigs) tryToPushConfigs(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if codebase.Status.Git == util.ProjectTemplatesPushedStatus {
		log.Info("Skip pushing templates. Templates already pushed")

		return nil
	}

	gitServer := &codebaseApi.GitServer{}
	if err := h.client.Get(
		ctx,
		client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace},
		gitServer,
	); err != nil {
		setFailedFields(codebase, codebaseApi.SetupDeploymentTemplates, err.Error())
		return fmt.Errorf("failed get GitServer: %w", err)
	}

	gitServerSecret := &corev1.Secret{}
	if err := h.client.Get(
		ctx,
		client.ObjectKey{Name: gitServer.Spec.NameSshKeySecret, Namespace: codebase.Namespace},
		gitServerSecret,
	); err != nil {
		return fmt.Errorf("failed to get GitServer secret: %w", err)
	}

	privateSSHKey := string(gitServerSecret.Data[util.PrivateSShKeyName])
	repoSshUrl := util.GetSSHUrl(gitServer, codebase.Spec.GetProjectID())
	wd := util.GetWorkDir(codebase.Name, codebase.Namespace)

	if !util.DoesDirectoryExist(wd) || util.IsDirectoryEmpty(wd) {
		log.Info("Start cloning repository", "url", repoSshUrl)

		if err := h.git.CloneRepositoryBySsh(
			ctx,
			privateSSHKey,
			gitServer.Spec.GitUser,
			repoSshUrl,
			wd,
			gitServer.Spec.SshPort,
		); err != nil {
			return fmt.Errorf("failed to clone git repository: %w", err)
		}

		log.Info("Repository has been cloned", "url", repoSshUrl)
	}

	if gitServer.Spec.GitProvider == codebaseApi.GitProviderGerrit {
		log.Info("Start adding commit hooks")

		if err := h.addCommitHooks(wd); err != nil {
			return fmt.Errorf("failed to add commit hooks: %w", err)
		}

		log.Info("Commit hooks have been added")
	}

	ru, err := util.GetRepoUrl(codebase)
	if err != nil {
		return fmt.Errorf("failed to build repo url: %w", err)
	}

	log.Info("Start checkout default branch", "branch", codebase.Spec.DefaultBranch, "repo", ru)

	err = CheckoutBranch(ru, wd, codebase.Spec.DefaultBranch, h.git, codebase, h.client)
	if err != nil {
		return fmt.Errorf("failed to checkout default branch %v in put_deploy_config has been failed: %w", codebase.Spec.DefaultBranch, err)
	}

	log.Info("Default branch has been checked out", "branch", codebase.Spec.DefaultBranch, "repo", ru)
	log.Info("Start preparing templates")

	err = template.PrepareTemplates(ctx, h.client, codebase, wd)
	if err != nil {
		return fmt.Errorf("failed to prepare template: %w", err)
	}

	log.Info("Templates have been prepared")
	log.Info("Start committing changes")

	err = h.git.CommitChanges(wd, fmt.Sprintf("Add deployment templates for %s", codebase.Name))
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Info("Changes have been committed")
	log.Info("Start pushing changes")

	err = h.git.PushChanges(privateSSHKey, gitServer.Spec.GitUser, wd, gitServer.Spec.SshPort, "--all")
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info("Changes have been pushed")

	if err = updateGitStatusWithPatch(ctx, h.client, codebase, codebaseApi.SetupDeploymentTemplates, util.ProjectTemplatesPushedStatus); err != nil {
		return err
	}

	log.Info("Config has been pushed")

	return nil
}

func (*PutDeployConfigs) addCommitHooks(wd string) error {
	destinationPath := fmt.Sprintf("%v/.git/hooks", wd)

	if err := util.CreateDirectory(destinationPath); err != nil {
		return fmt.Errorf("failed to create folder %v: %w", destinationPath, err)
	}

	assetsDir, err := util.GetAssetsDir()
	if err != nil {
		return fmt.Errorf("failed to get assets dir: %w", err)
	}

	fileName := "commit-msg"
	src := fmt.Sprintf("%v/configs/%v", assetsDir, fileName)
	dest := fmt.Sprintf("%v/%v", destinationPath, fileName)

	if err := util.CopyFile(src, dest); err != nil {
		return fmt.Errorf("failed to copy file %v: %w", fileName, err)
	}

	return nil
}
