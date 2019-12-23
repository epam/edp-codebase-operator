package codebase

import (
	"context"
	"errors"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/platform"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/git"
	"github.com/epmd-edp/codebase-operator/v2/pkg/jenkins"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	ClientSet "github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/perf"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/git_server"
	openshift_service "github.com/epmd-edp/codebase-operator/v2/pkg/service/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/settings"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/codebase-operator/v2/pkg/vcs"
	errWrap "github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"net/url"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
	"strings"
	"time"
)

var zlog = logf.Log.WithName("codebase-service")

type CodebaseService struct {
	CustomResource   *edpv1alpha1.Codebase
	Client           client.Client
	Scheme           *runtime.Scheme
	ClientSet        *openshift.ClientSet
	GitServerService git_server.GitServerService
	OpenshiftService openshift_service.OpenshiftService
}

const (
	Application         = "application"
	Autotests           = "autotests"
	ImportStrategy      = "import"
	KeyName             = "id_rsa"
	PipelinesSourcePath = "/usr/local/bin/pipelines"
	StatusFailed        = "failed"
	StatusFinished      = "created"
	StatusInProgress    = "in progress"
	GerritGitServerName = "gerrit"
	OtherLanguage       = "other"

	OpenshiftTemplate = "openshift-template"

	GithubDomain = "https://github.com/epmd-edp"
)

func (s CodebaseService) Create() error {
	if s.CustomResource.Status.Status != model.StatusInit {
		zlog.Info("Codebase is not in initialized status. Skipped.", "name", s.CustomResource.Name,
			"status", s.CustomResource.Status.Status)
		return nil
	}

	sp := s.CustomResource.Spec
	zlog.Info("Creating codebase...", "name", s.CustomResource.Name, "type", sp.Type)
	zlog.Info("Data Codebase", "name", s.CustomResource.Name, "language", sp.Lang,
		"description", sp.Description, "framework", sp.Framework, "build tool", sp.BuildTool,
		"strategy", sp.Strategy, "repository", sp.Repository, "database", sp.Database,
		"test report framework", sp.TestReportFramework, "type", sp.Type, "git server", sp.GitServer,
		"git url path", sp.GitUrlPath, "jenkins slave", sp.JenkinsSlave,
		"job provisioning", sp.JobProvisioning, "deployment sprint", sp.DeploymentScript)

	statusCR := edpv1alpha1.CodebaseStatus{
		Status:          StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Action:          edpv1alpha1.AcceptCodebaseRegistration,
		Result:          edpv1alpha1.Success,
		Username:        "system",
		Value:           "inactive",
	}

	err := updateStatusFields(s, statusCR)
	if err != nil {
		return errWrap.Wrap(err, "Error has been occurred in status update")
	}
	log.Println("Status of codebase CR has been changed to 'in progress'")

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	if err := settings.CreateWorkdir(wd); err != nil {
		return err
	}

	if sp.Strategy == ImportStrategy {
		if err := s.pushBuildConfigs(); err != nil {
			setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
			return errWrap.Wrap(err, "an error has been occurred while pushing build configs")
		}
	} else {
		gs, err := settings.GetGerritSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
		if err != nil {
			return err
		}

		us, err := settings.GetUserSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
		if err != nil {
			return err
		}

		if err := s.gerritConfiguration(us, gs); err != nil {
			setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
			return errWrap.Wrap(err, "Error has been occurred in gerrit configuration")
		}
		setIntermediateSuccessFields(s, edpv1alpha1.GerritRepositoryProvisioning)
		log.Println("Gerrit has been configured")

		if err := s.trySetupPerf(us); err != nil {
			return err
		}

		config, err := gerrit.ConfigInit(us.DnsWildcard, gs.SshPort, *s.CustomResource)
		if err != nil {
			return err
		}

		if err := gerrit.PushConfigs(*s.CustomResource, gs.SshPort, *config, *s.ClientSet); err != nil {
			setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
			return err
		}

		log.Println("Pipelines and templates has been pushed to Gerrit")
	}

	gs, err := s.getGitServerCR()
	if err != nil {
		return err
	}

	sshLink := s.generateSshLink(gs)
	log.Printf("Repository path: %v", sshLink)

	j, err := s.getJenkinsData(err)
	if err != nil {
		return err
	}

	err = s.triggerJobProvisioning(*j,
		map[string]string{
			"PARAM":                 "true",
			"NAME":                  s.CustomResource.Name,
			"BUILD_TOOL":            strings.ToLower(s.CustomResource.Spec.BuildTool),
			"GIT_SERVER_CR_NAME":    gs.Name,
			"GIT_SERVER_CR_VERSION": "v2",
			"GIT_CREDENTIALS_ID":    gs.NameSshKeySecret,
			"REPOSITORY_PATH":       sshLink,
		})
	if err != nil {
		setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
		return errWrap.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	setIntermediateSuccessFields(s, edpv1alpha1.JenkinsConfiguration)
	log.Println("Job provisioning has been triggered")

	err = os.RemoveAll(wd)
	if err != nil {
		return err
	}
	log.Printf("Workdir %v has been cleaned", wd)

	s.CustomResource.Status = edpv1alpha1.CodebaseStatus{
		Status:          StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupDeploymentTemplates,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}

	return nil
}

func (s CodebaseService) getJenkinsData(err error) (*model.Jenkins, error) {
	jen, err := settings.GetJenkins(s.Client, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	jt, ju, err := settings.GetJenkinsCreds(*jen, *s.ClientSet,
		s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	jurl := settings.GetJenkinsUrl(*jen, s.CustomResource.Namespace)

	return &model.Jenkins{
		JenkinsUrl:      jurl,
		JenkinsUsername: ju,
		JenkinsToken:    jt,
		JobName:         s.CustomResource.Spec.JobProvisioning,
	}, nil
}

func (s CodebaseService) generateSshLink(gs *model.GitServer) string {
	return fmt.Sprintf("ssh://%v@%v:%v%v", gs.GitUser, gs.GitHost, gs.SshPort, s.getRepositoryPath())
}

func (s CodebaseService) getGitServerCR() (*model.GitServer, error) {
	gitSec, err := s.OpenshiftService.GetGitServer(s.CustomResource.Spec.GitServer, s.CustomResource.Namespace)
	if err != nil {
		return nil, errWrap.Wrapf(err, "an error has occurred while getting Git Server CR for %v codebase",
			s.CustomResource.Name)
	}

	gs, err := model.ConvertToGitServer(*gitSec)
	if err != nil {
		return nil, errWrap.Wrapf(err, "an error has occurred while converting request Git Server to DTO for %v codebase",
			s.CustomResource.Name)
	}

	return gs, nil
}

func (s CodebaseService) pushBuildConfigs() error {
	log.Println("Start pushing build configs to remote git server...")

	gs, err := s.getGitServerCR()
	if err != nil {
		return err
	}

	secret, err := s.OpenshiftService.GetSecret(gs.NameSshKeySecret, s.CustomResource.Namespace)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred  while getting %v secret", gs.NameSshKeySecret))
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	templatesDir := fmt.Sprintf("%v/%v", wd, getTemplateFolderName(s.CustomResource.Spec.DeploymentScript))

	if err := util.CreateDirectory(templatesDir); err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while creating folder %v", templatesDir))
	}

	pathToCopiedGitFolder := fmt.Sprintf("%v/%v", templatesDir, s.CustomResource.Name)

	log.Printf("Path to local Git folder: %v", pathToCopiedGitFolder)

	repoData := collectRepositoryData(*gs, secret, *s.CustomResource.Spec.GitUrlPath, pathToCopiedGitFolder)

	log.Printf("Repo data is collected: repo url %v; port %v; user %v", repoData.RepositoryUrl, repoData.Port, repoData.User)

	err = s.GitServerService.CloneRepository(repoData)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while cloning repository %v", repoData.RepositoryUrl))
	}

	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", fmt.Sprintf("%v/%v", wd, getTemplateFolderName(s.CustomResource.Spec.DeploymentScript)),
		s.CustomResource.Name)

	if err := util.CreateDirectory(appTemplatesDir); err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while creating template folder %v", templatesDir))
	}

	if s.CustomResource.Spec.Type == Application {
		tc, err := s.buildTemplateConfig()
		if err != nil {
			return errWrap.Wrap(err, "an error has occurred while building template config")
		}

		if err := util.CopyTemplate(*tc); err != nil {
			return errWrap.Wrap(err, "an error has occurred while copying template")
		}
	}

	if err := util.CopyPipelines(s.CustomResource.Spec.Type, PipelinesSourcePath, pathToCopiedGitFolder); err != nil {
		return errWrap.Wrap(err, "an error has occurred while copying pipelines")
	}

	if err := s.GitServerService.CommitChanges(pathToCopiedGitFolder); err != nil {
		return errWrap.Wrap(err, "an error has occurred while commiting changes")
	}

	if err := s.GitServerService.PushChanges(repoData, pathToCopiedGitFolder); err != nil {
		return errWrap.Wrap(err, "an error has occurred while pushing changes")
	}

	if err := s.trySetupS2I(); err != nil {
		return err
	}

	log.Println("End pushing build configs to remote git server...")
	return nil
}

func getTemplateFolderName(deploymentScript string) string {
	if deploymentScript == util.HelmChartDeploymentScriptType {
		return "helm-charts"
	}
	return "oc-templates"
}

func (s CodebaseService) trySetupS2I() error {
	if s.CustomResource.Spec.Type != Application || strings.ToLower(s.CustomResource.Spec.Lang) == OtherLanguage {
		return nil
	}
	if platform.IsK8S() {
		return nil
	}
	is, err := gerrit.GetAppImageStream(strings.ToLower(s.CustomResource.Spec.Lang))
	if err != nil {
		return err
	}
	return gerrit.CreateS2IImageStream(*s.ClientSet, s.CustomResource.Name, s.CustomResource.Namespace, is)
}

func (s CodebaseService) buildTemplateConfig() (*model.GerritConfigGoTemplating, error) {
	log.Print("Start configuring template config ...")

	us, err := settings.GetUserSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}

	sp := s.CustomResource.Spec
	config := model.GerritConfigGoTemplating{
		Lang:             sp.Lang,
		Framework:        sp.Framework,
		BuildTool:        sp.BuildTool,
		DeploymentScript: s.CustomResource.Spec.DeploymentScript,
		WorkDir: fmt.Sprintf("/home/codebase-operator/edp/%v/%v",
			s.CustomResource.Namespace, s.CustomResource.Name),
		DnsWildcard: us.DnsWildcard,
		Name:        s.CustomResource.Name,
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

	log.Print("Gerrit config has been initialized")

	return &config, nil
}

func (s CodebaseService) triggerJobProvisioning(data model.Jenkins, parameters map[string]string) error {
	log.Printf("Start triggering job provision for %v", s.CustomResource.Name)

	jenkinsClient, err := jenkins.Init(data.JenkinsUrl, data.JenkinsUsername, data.JenkinsToken)
	if err != nil {
		return err
	}

	err = jenkinsClient.TriggerJobProvisioning(data.JobName, parameters, 10*time.Second, 12)
	if err != nil {
		return err
	}

	log.Printf("End triggering job provision for %v", s.CustomResource.Name)

	return nil
}

func collectRepositoryData(gitServer model.GitServer, secret *v1.Secret, projectPath, targetFolder string) model.RepositoryData {
	return model.RepositoryData{
		User:          gitServer.GitUser,
		Key:           string(secret.Data[KeyName]),
		Port:          gitServer.SshPort,
		RepositoryUrl: fmt.Sprintf("%v:%v%v", gitServer.GitHost, gitServer.SshPort, projectPath),
		FolderToClone: targetFolder,
	}
}

func (s CodebaseService) getRepositoryPath() string {
	if s.CustomResource.Spec.Strategy == ImportStrategy {
		return *s.CustomResource.Spec.GitUrlPath
	}
	return "/" + s.CustomResource.Name
}

func (s CodebaseService) gerritConfiguration(us *model.UserSettings, gs *model.GerritSettings) error {
	log.Printf("Start gerrit configuration for codebase: %v...", s.CustomResource.Name)

	gprk, _, err := settings.GetGerritCredentials(*s.ClientSet, s.CustomResource.Namespace)
	if err != nil {
		return err
	}

	log.Printf("Start creation of gerrit private key for codebase: %v...", s.CustomResource)

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	path := fmt.Sprintf("%v/gerrit-private.key", wd)
	if err := settings.CreateGerritPrivateKey(gprk, path); err != nil {
		log.Printf("Creation of gerrit private key for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}

	log.Printf("Start creation of ssh config for codebase: %v...", s.CustomResource.Name)
	sshConf, err := s.getSshConfig(us, gs)
	if err != nil {
		return err
	}

	if err := settings.CreateSshConfig(*sshConf); err != nil {
		log.Printf("Creation of ssh config for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}
	log.Printf("Start setup repo url for codebase: %v...", s.CustomResource.Name)

	repoUrl, err := getRepoUrl(GithubDomain, s.CustomResource.Spec)

	if err != nil {
		log.Printf("Setup repo url for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}

	log.Printf("Repository URL to clone sources has been retrieved: %v", *repoUrl)

	repositoryUsername, repositoryPassword, err := tryGetRepositoryCredentials(s, s.ClientSet)

	isRepositoryAccessible := git.CheckPermissions(*repoUrl, repositoryUsername, repositoryPassword)
	if !isRepositoryAccessible {
		return fmt.Errorf("user %v cannot get access to the repository %v", repositoryUsername, *repoUrl)
	}

	log.Printf("Start creation project in VCS for codebase: %v...", s.CustomResource.Name)

	if err := s.tryCreateProjectInVcs(us); err != nil {
		log.Printf("Creation project in VCS for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}
	log.Printf("Start clone project for codebase: %v...", s.CustomResource.Name)

	if err := s.tryCloneRepo(*repoUrl, repositoryUsername, repositoryPassword); err != nil {
		log.Printf("Clone project for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}

	log.Printf("Start creation project in Gerrit for codebase: %v...", s.CustomResource.Name)

	gf, err := s.getGerritConfig(gs)
	if err != nil {
		return err
	}

	if err := s.createProjectInGerrit(gf); err != nil {
		log.Printf("Creation project in Gerrit for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}

	log.Printf("Start push project to Gerrit for codebase: %v...", s.CustomResource.Name)
	if err := s.pushToGerrit(gf); err != nil {
		log.Printf("Push to gerrit for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}
	log.Printf("Start setup Gerrit replication for codebase: %v...", s.CustomResource.Name)

	if err := s.trySetupGerritReplication(us, gf); err != nil {
		log.Printf("Setup gerrit replication for codebase %v has been failed. Return error", s.CustomResource.Name)
		return err
	}
	log.Printf("Gerrit configuration has been finished successfully for codebase: %v...", s.CustomResource.Name)
	return nil
}

func (s CodebaseService) getVcsConfig(us *model.UserSettings) (*model.Vcs, error) {
	vcsGroupNameUrl, err := url.Parse(us.VcsGroupNameUrl)
	if err != nil {
		return nil, err
	}

	projectVcsHostnameUrl := vcsGroupNameUrl.Scheme + "://" + vcsGroupNameUrl.Host
	VcsCredentialsSecretName := "vcs-autouser-codebase-" + s.CustomResource.Name + "-temp"
	vcsAutoUserLogin, vcsAutoUserPassword, err := settings.GetVcsBasicAuthConfig(*s.ClientSet,
		s.CustomResource.Namespace, VcsCredentialsSecretName)

	vcsTool, err := vcs.CreateVCSClient(us.VcsToolName, projectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		return nil, err
	}

	vcsSshUrl, err := vcsTool.GetRepositorySshUrl(vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)], s.CustomResource.Name)
	if err != nil {
		return nil, err
	}

	return &model.Vcs{
		VcsSshUrl:             vcsSshUrl,
		VcsIntegrationEnabled: us.VcsIntegrationEnabled,
		VcsToolName:           us.VcsToolName,
		ProjectVcsHostnameUrl: projectVcsHostnameUrl,
		ProjectVcsGroupPath:   vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)],
	}, nil
}

func (s CodebaseService) getGerritConfig(gs *model.GerritSettings) (*model.GerritConf, error) {
	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	return &model.GerritConf{
		GerritKeyPath: fmt.Sprintf("%v/gerrit-private.key", wd),
		GerritHost:    fmt.Sprintf("gerrit.%v", s.CustomResource.Namespace),
		SshPort:       gs.SshPort,
		WorkDir:       wd,
	}, nil
}

func (s CodebaseService) getSshConfig(us *model.UserSettings, gs *model.GerritSettings) (*model.SshConfig, error) {
	vcsGroup, err := url.Parse(us.VcsGroupNameUrl)
	if err != nil {
		return nil, err
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	return &model.SshConfig{
		CicdNamespace:         s.CustomResource.Namespace,
		SshPort:               gs.SshPort,
		GerritKeyPath:         fmt.Sprintf("%v/gerrit-private.key", wd),
		VcsIntegrationEnabled: us.VcsIntegrationEnabled,
		ProjectVcsHostname:    vcsGroup.Host,
		VcsSshPort:            us.VcsSshPort,
		VcsKeyPath:            wd + "/vcs-private.key",
	}, nil
}
func tryGetRepositoryCredentials(s CodebaseService, clientSet *ClientSet.ClientSet) (string, string, error) {
	if s.CustomResource.Spec.Repository != nil {
		return getRepoCreds(s, clientSet)
	}
	return "", "", nil
}

func getRepoCreds(s CodebaseService, clientSet *ClientSet.ClientSet) (string, string, error) {
	repositoryCredentialsSecretName := fmt.Sprintf("repository-codebase-%v-temp", s.CustomResource.Name)
	repositoryUsername, repositoryPassword, err := settings.GetVcsBasicAuthConfig(*clientSet,
		s.CustomResource.Namespace, repositoryCredentialsSecretName)
	if err != nil {
		log.Printf("Unable to get VCS credentials from secret %v", repositoryCredentialsSecretName)
		return "", "", err
	}
	return repositoryUsername, repositoryPassword, nil
}

func (s CodebaseService) trySetupGerritReplication(us *model.UserSettings, gf *model.GerritConf) error {
	if us.VcsIntegrationEnabled {
		vcsConf, err := s.getVcsConfig(us)
		if err != nil {
			return err
		}
		return gerrit.SetupProjectReplication(s.CustomResource.Name, s.CustomResource.Namespace, *gf, *vcsConf, *s.ClientSet)
	}

	log.Print("Skipped gerrit replication configuration. VCS integration isn't enabled")
	return nil
}

func (s CodebaseService) trySetupPerf(us *model.UserSettings) error {
	if us.PerfIntegrationEnabled {
		return s.setupPerf(us)
	}
	log.Print("Skipped perf configuration. Perf integration isn't enabled")
	return nil
}

func (s CodebaseService) setupPerf(us *model.UserSettings) error {
	log.Println("Start perf configuration...")
	perfSetting := perf.GetPerfSettings(*s.ClientSet, s.CustomResource.Namespace)
	log.Printf("Perf setting have been retrieved: %v", perfSetting)
	secret := perf.GetPerfCredentials(*s.ClientSet, s.CustomResource.Namespace)

	perfUrl := perfSetting[perf.UrlSettingsKey]
	user := string(secret["username"])
	log.Printf("Username for perf integration has been retrieved: %v", user)
	pass := string(secret["password"])
	perfClient, err := perf.NewRestClient(perfUrl, user, pass)
	if err != nil {
		log.Printf("Error has occurred during perf client init: %v", err)
		return err
	}

	err = setupJenkinsPerf(perfClient, s.CustomResource.Name, perfSetting[perf.JenkinsDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Jenkins Perf: %v", err)
		return err
	}

	err = setupSonarPerf(perfClient, s.CustomResource.Name, perfSetting[perf.SonarDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Sonar Perf: %v", err)
		return err
	}

	err = setupGerritPerf(perfClient, s.CustomResource.Name, perfSetting[perf.GerritDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Gerrit Perf: %v", err)
		return err
	}

	err = s.trySetupGitlabPerf(perfClient, perfSetting[perf.GitlabDsSettingsKey], us)
	if err != nil {
		log.Printf("Error has occurred during setup Gitlab Perf: %v", err)
		return err
	}

	log.Println("Perf integration has been successfully finished")
	return nil
}

func setupJenkinsPerf(client *perf.Client, codebaseName string, dsId string) error {
	jenkinsDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}
	jenkinsJobs := []string{fmt.Sprintf("/Code-review-%s", codebaseName), fmt.Sprintf("/Build-%s", codebaseName)}
	return client.AddJobsToJenkinsDS(jenkinsDsID, jenkinsJobs)
}

func setupSonarPerf(client *perf.Client, codebaseName string, dsId string) error {
	sonarDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}
	sonarProjects := []string{fmt.Sprintf("%s:master", codebaseName)}
	return client.AddProjectsToSonarDS(sonarDsID, sonarProjects)
}

func setupGerritPerf(client *perf.Client, codebaseName string, dsId string) error {
	gerritDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}

	gerritProjects := []perf.GerritPerfConfig{{ProjectName: codebaseName, Branches: []string{"master"}}}
	return client.AddProjectsToGerritDS(gerritDsID, gerritProjects)
}

func (s CodebaseService) trySetupGitlabPerf(client *perf.Client, dsId string, us *model.UserSettings) error {
	if isGitLab(us.VcsIntegrationEnabled, us.VcsToolName) {
		vcsGroupNameUrl, err := url.Parse(us.VcsGroupNameUrl)
		if err != nil {
			return err
		}
		vcspp := vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)] + "/" + s.CustomResource.Name
		return setupGitlabPerf(client, vcspp, dsId)
	}
	return nil
}

func setupGitlabPerf(client *perf.Client, codebaseName string, dsId string) error {
	gitDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}

	gitProjects := map[string]string{codebaseName: "master"}
	return client.AddRepositoriesToGitlabDS(gitDsID, gitProjects)
}

func isGitLab(vcsIntegrationEnabled bool, vcsToolName model.VCSTool) bool {
	return vcsIntegrationEnabled && vcsToolName == model.GitLab
}

func (s CodebaseService) createProjectInGerrit(conf *model.GerritConf) error {
	projectExist, err := gerrit.CheckProjectExist(conf.GerritKeyPath, conf.GerritHost, conf.SshPort, s.CustomResource.Name)
	if err != nil {
		return err
	}
	if *projectExist {
		return errors.New("couldn't create project in Gerrit. Project already exists")
	}

	if err := gerrit.CreateProject(conf.GerritKeyPath, conf.GerritHost, conf.SshPort, s.CustomResource.Name); err != nil {
		return err
	}
	return nil
}

func (s CodebaseService) pushToGerrit(conf *model.GerritConf) error {
	if err := gerrit.AddRemoteLinkToGerrit(conf.WorkDir+"/"+s.CustomResource.Name, conf.GerritHost, conf.SshPort, s.CustomResource.Name); err != nil {
		return err
	}

	if err := gerrit.PushToGerrit(conf.WorkDir+"/"+s.CustomResource.Name, conf.GerritKeyPath); err != nil {
		return err
	}

	return nil
}

func (s *CodebaseService) tryCreateProjectInVcs(us *model.UserSettings) error {
	if us.VcsIntegrationEnabled {
		if err := s.createProjectInVcs(us); err != nil {
			return err
		}
		return nil
	}

	log.Println("VCS integration isn't enabled")
	return nil
}

func (s *CodebaseService) createProjectInVcs(us *model.UserSettings) error {
	vcsConf, err := s.getVcsConfig(us)
	if err != nil {
		return err
	}

	VcsCredentialsSecretName := "vcs-autouser-codebase-" + s.CustomResource.Name + "-temp"
	vcsAutoUserLogin, vcsAutoUserPassword, err := settings.GetVcsBasicAuthConfig(*s.ClientSet,
		s.CustomResource.Namespace, VcsCredentialsSecretName)

	vcsTool, err := vcs.CreateVCSClient(model.VCSTool(vcsConf.VcsToolName),
		vcsConf.ProjectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		log.Printf("Unable to create VCS client: %v", err)
		return err
	}

	projectExist, err := vcsTool.CheckProjectExist(vcsConf.ProjectVcsGroupPath, s.CustomResource.Name)
	if err != nil {
		return err
	}
	if *projectExist {
		return errors.New("Couldn't copy project to your VCS group. Repository %s is already exists in " +
			s.CustomResource.Name + "" + vcsConf.ProjectVcsGroupPath)
	} else {
		_, err = vcsTool.CreateProject(vcsConf.ProjectVcsGroupPath, s.CustomResource.Name)
		if err != nil {
			return err
		}
		vcsConf.VcsSshUrl, err = vcsTool.GetRepositorySshUrl(vcsConf.ProjectVcsGroupPath, s.CustomResource.Name)
	}
	return err
}

func (s CodebaseService) tryCloneRepo(repoUrl string, repositoryUsername string, repositoryPassword string) error {
	workDir := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	destination := fmt.Sprintf("%v/%v", workDir, s.CustomResource.Name)

	if err := git.CloneRepo(repoUrl, repositoryUsername, repositoryPassword, destination); err != nil {
		return err
	}
	log.Printf("Repository has been cloned to %v", destination)

	return nil
}

func setFailedFields(s CodebaseService, action edpv1alpha1.ActionType, message string) {
	s.CustomResource.Status = edpv1alpha1.CodebaseStatus{
		Status:          StatusFailed,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}
}

func setIntermediateSuccessFields(s CodebaseService, action edpv1alpha1.ActionType) {
	s.CustomResource.Status = edpv1alpha1.CodebaseStatus{
		Status:          StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          edpv1alpha1.Success,
		Value:           "inactive",
	}
	err := s.Client.Status().Update(context.TODO(), s.CustomResource)
	if err != nil {
		err = s.Client.Update(context.TODO(), s.CustomResource)
		if err != nil {
			log.Printf("Error has been occured during the update success fields fot codebase: %v", s.CustomResource.Name)
		}
	}
}

func updateStatusFields(service CodebaseService, status edpv1alpha1.CodebaseStatus) error {
	service.CustomResource.Status = status
	err := service.Client.Status().Update(context.TODO(), service.CustomResource)
	if err != nil {
		err := service.Client.Update(context.TODO(), service.CustomResource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s CodebaseService) Update() {

}

func (s CodebaseService) Delete() {

}
