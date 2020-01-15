package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"os"
	"strings"
	"text/template"
)

type PutDeployConfigs struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h PutDeployConfigs) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start pushing configs...")

	gs, us, err := util.GetConfigSettings(h.clientSet.CoreClient, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "unable get config settings")
	}

	config, err := configInit(us.DnsWildcard, gs.SshPort, *c)
	if err != nil {
		return err
	}

	if err := h.tryToPushConfigs(*c, gs.SshPort, config); err != nil {
		setFailedFields(*c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs", "codebase name", c.Name)
	}

	if err := h.tryToCreateImageStream(*c); err != nil {
		setFailedFields(*c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't create IS", "codebase name", c.Name)
	}
	rLog.Info("end pushing configs")
	return nextServeOrNil(h.next, c)
}

func configInit(dnsWildcard string, sshPort int32, codebase edpv1alpha1.Codebase) (model.GerritConfigGoTemplating, error) {
	log.Info("start creating Gerrit config")
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", codebase.Namespace, codebase.Name)
	cf := model.GerritConfigGoTemplating{
		Type:             codebase.Spec.Type,
		DnsWildcard:      dnsWildcard,
		Name:             codebase.Name,
		DeploymentScript: codebase.Spec.DeploymentScript,
		WorkDir:          wd,
		Lang:             codebase.Spec.Lang,
		Framework:        codebase.Spec.Framework,
		BuildTool:        codebase.Spec.BuildTool,
		TemplatesDir:     createTemplateDirectory(wd, codebase.Spec.DeploymentScript),
		CloneSshUrl:      fmt.Sprintf("ssh://project-creator@gerrit.%v:%v/%v", codebase.Namespace, sshPort, codebase.Name),
	}
	if codebase.Spec.Repository != nil {
		cf.RepositoryUrl = &codebase.Spec.Repository.Url
	}
	if codebase.Spec.Database != nil {
		cf.Database = codebase.Spec.Database
	}
	if codebase.Spec.Route != nil {
		cf.Route = codebase.Spec.Route
	}
	log.Info("Gerrit config has been initialized")
	return cf, nil
}

func createTemplateDirectory(workDir string, deploymentScriptType string) string {
	if deploymentScriptType == util.OpenshiftTemplate {
		return fmt.Sprintf("%v/%v", workDir, util.OcTemplatesFolder)
	}
	return fmt.Sprintf("%v/%v", workDir, util.HelmChartTemplatesFolder)
}

func (h PutDeployConfigs) tryToPushConfigs(c edpv1alpha1.Codebase, sshPort int32, config model.GerritConfigGoTemplating) error {
	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", config.TemplatesDir, c.Name)
	appConfigFilesDir := fmt.Sprintf("%v/%v/config-files", config.TemplatesDir, c.Name)

	s, err := util.GetSecret(*h.clientSet.CoreClient, "gerrit-project-creator", c.Namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])

	if err := util.CreateDirectory(config.TemplatesDir); err != nil {
		return err
	}

	if err := cloneProjectRepoFromGerrit(sshPort, idrsa, c.Name, c.Namespace, config); err != nil {
		return err
	}

	if err := util.CreateDirectory(appConfigFilesDir); err != nil {
		return err
	}

	destinationPath := fmt.Sprintf("%v/%v/config-files", config.TemplatesDir, c.Name)
	fileName := "Readme.md"
	src := fmt.Sprintf("%v/%v", util.GerritTemplates, fileName)
	dest := fmt.Sprintf("%v/%v", destinationPath, fileName)
	if err := util.CopyFile(src, dest); err != nil {
		return err
	}

	if err := util.CreateDirectory(appTemplatesDir); err != nil {
		return err
	}

	if c.Spec.Type == "application" {
		if err := util.CopyTemplate(config); err != nil {
			return err
		}
	}

	dest = fmt.Sprintf("%v/%v", config.TemplatesDir, config.Name)
	if err := util.CopyPipelines(c.Spec.Type, util.PipelineTemplates, dest); err != nil {
		return nil
	}

	if strings.ToLower(config.Lang) == "javascript" {
		if err := copySonarConfigs(config); err != nil {
			return err
		}
	}

	d := fmt.Sprintf("%v/%v", config.TemplatesDir, c.Name)
	if err := git.CommitChanges(d); err != nil {
		return err
	}

	if err := git.PushChanges(idrsa, "project-creator", d); err != nil {
		return err
	}
	return nil
}

func copySonarConfigs(config model.GerritConfigGoTemplating) error {
	sonarConfigPath := fmt.Sprintf("%v/%v/sonar-project.properties", config.TemplatesDir, config.Name)
	log.Info("start copying sonar configs", "path", sonarConfigPath)
	if _, err := os.Stat(sonarConfigPath); err == nil {
		return nil
	}

	f, err := os.Create(sonarConfigPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := template.New("sonar-project.properties.tmpl").
		ParseFiles("/usr/local/bin/templates/sonar/sonar-project.properties.tmpl")
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, config); err != nil {
		return errors.Wrapf(err, "couldn't render Sonar configs fo JS app: %v", config.Name)
	}
	log.Info("Sonar configs has been copied", "codebase name", config.Name)
	return nil
}

func cloneProjectRepoFromGerrit(sshPort int32, idrsa, name, namespace string, config model.GerritConfigGoTemplating) error {
	log.Info("start cloning repository from Gerrit", "ssh url", config.CloneSshUrl)

	var (
		s *ssh.Session
		c *ssh.Client

		h = fmt.Sprintf("gerrit.%v", namespace)
	)

	sshcl, err := gerrit.SshInit(sshPort, idrsa, h)
	if err != nil {
		return errors.Wrap(err, "couldn't initialize SSH client")
	}

	if s, c, err = sshcl.NewSession(); err != nil {
		return errors.Wrap(err, "couldn't initialize SSH session")
	}

	defer func() {
		if deferErr := s.Close(); deferErr != nil {
			err = deferErr
		}
		if deferErr := c.Close(); deferErr != nil {
			err = deferErr
		}
	}()

	d := fmt.Sprintf("%v/%v", config.TemplatesDir, name)
	if err := git.CloneRepositoryBySsh(idrsa, "project-creator", config.CloneSshUrl, d); err != nil {
		return err
	}

	destinationPath := fmt.Sprintf("%v/%v/.git/hooks", config.TemplatesDir, name)
	if err := util.CreateDirectory(destinationPath); err != nil {
		return errors.Wrapf(err, "couldn't create folder %v", destinationPath)
	}

	sourcePath := "/usr/local/bin/configs"
	fileName := "commit-msg"
	src := fmt.Sprintf("%v/%v", sourcePath, fileName)
	dest := fmt.Sprintf("%v/%v", destinationPath, fileName)
	if err := util.CopyFile(src, dest); err != nil {
		return errors.Wrapf(err, "couldn't copy file %v", fileName)
	}
	return nil
}

func (h PutDeployConfigs) tryToCreateImageStream(c edpv1alpha1.Codebase) error {
	log.Info("start creating image stream", "codebase name", c.Name)
	if !isSupportedType(c) {
		log.Info("couldn't create image stream as type of codebase is not acceptable")
		return nil
	}

	appImageStream, err := util.GetAppImageStream(c.Spec.Lang)
	if err != nil {
		return err
	}
	return util.CreateS2IImageStream(*h.clientSet.ImageClient, c.Name, c.Namespace, appImageStream)
}

func isSupportedType(codebase edpv1alpha1.Codebase) bool {
	return codebase.Spec.Type == "application" && codebase.Spec.Lang != "other"
}
