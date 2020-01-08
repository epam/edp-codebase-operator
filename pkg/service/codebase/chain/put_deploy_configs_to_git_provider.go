package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/platform"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"strings"
)

type PutDeployConfigsToGitProvider struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

const (
	Application   = "application"
	OtherLanguage = "other"
)

func (h PutDeployConfigsToGitProvider) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start pushing configs...")

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", fmt.Sprintf("%v/%v", wd, getTemplateFolderName(c.Spec.DeploymentScript)),
		c.Name)
	if err := util.CreateDirectory(appTemplatesDir); err != nil {
		return errors.Wrapf(err, "an error has occurred while creating template folder %v", appTemplatesDir)
	}

	if c.Spec.Type == Application {
		tc, err := h.getTemplateConfig(*c)
		if err != nil {
			return errors.Wrapf(err, "an error has occurred while building template config for %v codebase", c.Name)
		}

		if err := util.CopyTemplate(*tc); err != nil {
			return errors.Wrapf(err, "an error has occurred while copying template for %v codebase", c.Name)
		}
	}

	templatesDir := fmt.Sprintf("%v/%v", wd, getTemplateFolderName(c.Spec.DeploymentScript))
	pathToCopiedGitFolder := fmt.Sprintf("%v/%v", templatesDir, c.Name)
	if err := util.CopyPipelines(c.Spec.Type, util.PipelineTemplates, pathToCopiedGitFolder); err != nil {
		return errors.Wrapf(err, "an error has occurred while copying pipelines for %v codebase", c.Name)
	}

	if err := git.CommitChanges(pathToCopiedGitFolder); err != nil {
		return errors.Wrapf(err, "an error has occurred while commiting changes for %v codebase", c.Name)
	}

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(*h.clientSet.CoreClient, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	td := fmt.Sprintf("%v/%v", wd, getTemplateFolderName(c.Spec.DeploymentScript))
	gf := fmt.Sprintf("%v/%v", td, c.Name)
	log.Info("path to local Git folder", "path", gf)

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := git.PushChanges(k, u, pathToCopiedGitFolder); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v codebase", c.Name)
	}

	if err := h.trySetupS2I(*c); err != nil {
		return err
	}
	rLog.Info("end pushing configs to remote git server")
	return nextServeOrNil(h.next, c)
}

func (h PutDeployConfigsToGitProvider) getTemplateConfig(c edpv1alpha1.Codebase) (*model.GerritConfigGoTemplating, error) {
	log.Info("Start configuring template config ...")
	_, us, err := util.GetConfigSettings(h.clientSet, c.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "unable get config settings")
	}

	sp := c.Spec
	config := model.GerritConfigGoTemplating{
		Lang:             sp.Lang,
		Framework:        sp.Framework,
		BuildTool:        sp.BuildTool,
		DeploymentScript: sp.DeploymentScript,
		WorkDir:          fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name),
		DnsWildcard:      us.DnsWildcard,
		Name:             c.Name,
	}
	if sp.Repository != nil {
		config.RepositoryUrl = &sp.Repository.Url
	}
	if sp.Database != nil {
		config.Database = sp.Database
	}
	if sp.Route != nil {
		config.Route = sp.Route
	}
	log.Info("Gerrit config has been initialized")
	return &config, nil
}

func (h PutDeployConfigsToGitProvider) trySetupS2I(c edpv1alpha1.Codebase) error {
	log.Info("start creating image stream", "codebase name", c.Name)
	if c.Spec.Type != Application || strings.ToLower(c.Spec.Lang) == OtherLanguage {
		return nil
	}
	if platform.IsK8S() {
		return nil
	}
	is, err := util.GetAppImageStream(strings.ToLower(c.Spec.Lang))
	if err != nil {
		return err
	}
	return util.CreateS2IImageStream(*h.clientSet.ImageClient, c.Name, c.Namespace, is)
}
