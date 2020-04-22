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

	cs, err := s.initCodebaseSettings()
	if err != nil {
		setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
		return errWrap.Wrap(err, "an error has been occurred while initializing codebase settings")
	}
	log.Printf("Codebase settings are set up for %v codebase", cs.Name)

	if cs.Strategy == ImportStrategy {
		err = s.pushBuildConfigs(cs)
		if err != nil {
			setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
			return errWrap.Wrap(err, "an error has been occurred while pushing build configs")
		}
	} else {
		err = s.gerritConfiguration(cs)
		if err != nil {
			setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
			return errWrap.Wrap(err, "Error has been occurred in gerrit configuration")
		}
		setIntermediateSuccessFields(s, edpv1alpha1.GerritRepositoryProvisioning)
		log.Println("Gerrit has been configured")

		err = s.trySetupPerf(*cs)

		config, err := gerrit.ConfigInit(*cs, s.CustomResource.Spec)
		err = gerrit.PushConfigs(*config, *cs, *s.ClientSet)
		if err != nil {
			setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
			return err
		}

		log.Println("Pipelines and templates has been pushed to Gerrit")
	}

	sshLink := s.generateSshLink(*cs)
	log.Printf("Repository path: %v", sshLink)

	err = s.triggerJobProvisioning(model.Jenkins{
		JenkinsUrl:      cs.JenkinsUrl,
		JenkinsUsername: cs.JenkinsUsername,
		JenkinsToken:    cs.JenkinsToken,
		JobName:         cs.JobProvisioning,
	},
		map[string]string{
			"PARAM":                 "true",
			"NAME":                  s.CustomResource.Name,
			"BUILD_TOOL":            strings.ToLower(s.CustomResource.Spec.BuildTool),
			"GIT_SERVER_CR_NAME":    cs.GitServer.Name,
			"GIT_SERVER_CR_VERSION": "v2",
			"GIT_CREDENTIALS_ID":    cs.GitServer.NameSshKeySecret,
			"REPOSITORY_PATH":       sshLink,
		})
	if err != nil {
		setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
		return errWrap.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	setIntermediateSuccessFields(s, edpv1alpha1.JenkinsConfiguration)
	log.Println("Job provisioning has been triggered")

	err = os.RemoveAll(cs.WorkDir)
	if err != nil {
		return err
	}
	log.Printf("Workdir %v has been cleaned", cs.WorkDir)

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

func (s CodebaseService) generateSshLink(codebaseSettings model.CodebaseSettings) string {
	return fmt.Sprintf("ssh://%v@%v:%v%v", codebaseSettings.GitServer.GitUser, codebaseSettings.GitServer.GitHost,
		codebaseSettings.GitServer.SshPort, codebaseSettings.RepositoryPath)
}

func (s CodebaseService) pushBuildConfigs(codebaseSettings *model.CodebaseSettings) error {
	log.Println("Start pushing build configs to remote git server...")

	gitServer := codebaseSettings.GitServer

	secret, err := s.OpenshiftService.GetSecret(gitServer.NameSshKeySecret, codebaseSettings.CicdNamespace)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred  while getting %v secret", gitServer.NameSshKeySecret))
	}

	templatesDir := fmt.Sprintf("%v/%v", codebaseSettings.WorkDir, getTemplateFolderName(codebaseSettings.DeploymentScript))
	err = util.CreateDirectory(templatesDir)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while creating folder %v", templatesDir))
	}

	pathToCopiedGitFolder := fmt.Sprintf("%v/%v", templatesDir, codebaseSettings.Name)

	log.Printf("Path to local Git folder: %v", pathToCopiedGitFolder)

	repoData := collectRepositoryData(gitServer, secret, *s.CustomResource.Spec.GitUrlPath, pathToCopiedGitFolder)

	log.Printf("Repo data is collected: repo url %v; port %v; user %v", repoData.RepositoryUrl, repoData.Port, repoData.User)

	err = s.GitServerService.CloneRepository(repoData)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while cloning repository %v", repoData.RepositoryUrl))
	}

	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", fmt.Sprintf("%v/%v", codebaseSettings.WorkDir, getTemplateFolderName(codebaseSettings.DeploymentScript)),
		codebaseSettings.Name)
	err = util.CreateDirectory(appTemplatesDir)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while creating template folder %v", templatesDir))
	}

	if codebaseSettings.Type == Application {
		templateConfig := s.buildTemplateConfig(*codebaseSettings)

		err = util.CopyTemplate(templateConfig)
		if err != nil {
			return errWrap.Wrap(err, "an error has occurred while copying template")
		}
	}

	err = util.CopyPipelines(codebaseSettings.Type, PipelinesSourcePath, pathToCopiedGitFolder)
	if err != nil {
		return errWrap.Wrap(err, "an error has occurred while copying pipelines")
	}

	err = s.GitServerService.CommitChanges(pathToCopiedGitFolder)
	if err != nil {
		return errWrap.Wrap(err, "an error has occurred while commiting changes")
	}

	err = s.GitServerService.PushChanges(repoData, pathToCopiedGitFolder)
	if err != nil {
		return errWrap.Wrap(err, "an error has occurred while pushing changes")
	}

	err = s.trySetupS2I(*codebaseSettings)
	if err != nil {
		return err
	}

	if err := util.RemoveDirectory(pathToCopiedGitFolder); err != nil {
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

func (s CodebaseService) trySetupS2I(cs model.CodebaseSettings) error {
	if cs.Type != Application || cs.Lang == OtherLanguage {
		return nil
	}
	if platform.IsK8S() {
		return nil
	}
	is, err := gerrit.GetAppImageStream(cs.Lang)
	if err != nil {
		return err
	}
	return gerrit.CreateS2IImageStream(*s.ClientSet, cs.Name, cs.CicdNamespace, is)
}

func (s CodebaseService) buildTemplateConfig(codebaseSettings model.CodebaseSettings) model.GerritConfigGoTemplating {
	log.Print("Start configuring template config ...")
	sp := s.CustomResource.Spec
	config := model.GerritConfigGoTemplating{
		Lang:             sp.Lang,
		Framework:        sp.Framework,
		BuildTool:        sp.BuildTool,
		CodebaseSettings: codebaseSettings,
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

	return config
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

func (s CodebaseService) initCodebaseSettings() (*model.CodebaseSettings, error) {
	var (
		workDir = fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	)

	err := settings.CreateWorkdir(workDir)
	if err != nil {
		return nil, err
	}

	cs := model.CodebaseSettings{}
	cs.WorkDir = workDir
	cs.Lang = strings.ToLower(s.CustomResource.Spec.Lang)
	cs.Name = s.CustomResource.Name
	cs.Type = s.CustomResource.Spec.Type
	cs.CicdNamespace = s.CustomResource.Namespace
	cs.RepositoryPath = s.getRepositoryPath()
	cs.JobProvisioning = s.CustomResource.Spec.JobProvisioning
	cs.Strategy = string(s.CustomResource.Spec.Strategy)
	cs.DeploymentScript = s.CustomResource.Spec.DeploymentScript

	if cs.Type == Application {
		cs.Framework = *s.CustomResource.Spec.Framework
	}

	jen, err := settings.GetJenkins(s.Client, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	cs.JenkinsToken, cs.JenkinsUsername, err = settings.GetJenkinsCreds(*jen, *s.ClientSet,
		s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	cs.JenkinsUrl = settings.GetJenkinsUrl(*jen, s.CustomResource.Namespace)

	userSettings, err := settings.GetUserSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	cs.UserSettings = *userSettings

	if cs.UserSettings.VcsIntegrationEnabled {
		VcsGroupNameUrl, err := url.Parse(cs.UserSettings.VcsGroupNameUrl)
		if err != nil {
			log.Print(err)
		}
		cs.ProjectVcsHostname = VcsGroupNameUrl.Host
		cs.ProjectVcsGroupPath = VcsGroupNameUrl.Path[1:len(VcsGroupNameUrl.Path)]
		cs.ProjectVcsHostnameUrl = VcsGroupNameUrl.Scheme + "://" + cs.ProjectVcsHostname
		cs.VcsProjectPath = cs.ProjectVcsGroupPath + "/" + s.CustomResource.Name
		cs.VcsKeyPath = cs.WorkDir + "/vcs-private.key"

		cs.VcsAutouserSshKey, cs.VcsAutouserEmail, err = settings.GetVcsCredentials(*s.ClientSet,
			s.CustomResource.Namespace)
	} else {
		log.Printf("VCS integration isn't enabled")
	}

	if s.CustomResource.Spec.Strategy != ImportStrategy {
		cs.BasicPatternUrl = "https://github.com/epmd-edp"
		cs.GerritHost = fmt.Sprintf("gerrit.%v", cs.CicdNamespace)
		cs.GerritKeyPath = fmt.Sprintf("%v/gerrit-private.key", cs.WorkDir)

		gerritSettings, err := settings.GetGerritSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
		if err != nil {
			return nil, err
		}
		cs.GerritSettings = *gerritSettings

		cs.GerritPrivateKey, cs.GerritPublicKey, err = settings.GetGerritCredentials(*s.ClientSet,
			s.CustomResource.Namespace)
	}

	gitServerRequest, err := s.OpenshiftService.GetGitServer(s.CustomResource.Spec.GitServer, cs.CicdNamespace)
	if err != nil {
		return nil, errWrap.Wrapf(err, "an error has occurred while getting Git Server CR for %v codebase",
			cs.Name)
	}

	gitServer, err := model.ConvertToGitServer(*gitServerRequest)
	if err != nil {
		return nil, errWrap.Wrapf(err, "an error has occurred while converting request Git Server to DTO for %v codebase",
			cs.Name)
	}
	cs.GitServer = *gitServer

	log.Printf("Retrieving settings has been finished.")

	return &cs, nil
}

func (s CodebaseService) getRepositoryPath() string {
	if s.CustomResource.Spec.Strategy == ImportStrategy {
		return *s.CustomResource.Spec.GitUrlPath
	}
	return "/" + s.CustomResource.Name
}

func (s CodebaseService) gerritConfiguration(codebaseSettings *model.CodebaseSettings) error {
	log.Printf("Start gerrit configuration for codebase: %v...", codebaseSettings.Name)

	log.Printf("Start creation of gerrit private key for codebase: %v...", codebaseSettings.Name)
	err := settings.CreateGerritPrivateKey(codebaseSettings.GerritPrivateKey, codebaseSettings.GerritKeyPath)
	if err != nil {
		log.Printf("Creation of gerrit private key for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Start creation of ssh config for codebase: %v...", codebaseSettings.Name)
	err = settings.CreateSshConfig(*codebaseSettings)
	if err != nil {
		log.Printf("Creation of ssh config for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Start setup repo url for codebase: %v...", codebaseSettings.Name)

	repoUrl, err := getRepoUrl(codebaseSettings.BasicPatternUrl, s.CustomResource.Spec)

	if err != nil {
		log.Printf("Setup repo url for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}

	log.Printf("Repository URL to clone sources has been retrieved: %v", *repoUrl)

	repositoryUsername, repositoryPassword, err := tryGetRepositoryCredentials(s, s.ClientSet)

	isRepositoryAccessible := git.CheckPermissions(*repoUrl, repositoryUsername, repositoryPassword)
	if !isRepositoryAccessible {
		return fmt.Errorf("user %v cannot get access to the repository %v", repositoryUsername, *repoUrl)
	}
	log.Printf("Start creation project in VCS for codebase: %v...", codebaseSettings.Name)
	err = tryCreateProjectInVcs(codebaseSettings, &s, *s.ClientSet)
	if err != nil {
		log.Printf("Creation project in VCS for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Start clone project for codebase: %v...", codebaseSettings.Name)
	err = tryCloneRepo(s, *codebaseSettings, *repoUrl, repositoryUsername, repositoryPassword)
	if err != nil {
		log.Printf("Clone project for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Start creation project in Gerrit for codebase: %v...", codebaseSettings.Name)
	err = createProjectInGerrit(codebaseSettings, &s)
	if err != nil {
		log.Printf("Creation project in Gerrit for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Start push project to Gerrit for codebase: %v...", codebaseSettings.Name)
	err = pushToGerrit(codebaseSettings, &s)
	if err != nil {
		log.Printf("Push to gerrit for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Start setup Gerrit replication for codebase: %v...", codebaseSettings.Name)
	err = trySetupGerritReplication(*codebaseSettings, *s.ClientSet)
	if err != nil {
		log.Printf("Setup gerrit replication for codebase %v has been failed. Return error", codebaseSettings.Name)
		return err
	}
	log.Printf("Gerrit configuration has been finished successfully for codebase: %v...", codebaseSettings.Name)
	return nil
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

func trySetupGerritReplication(codebaseSettings model.CodebaseSettings, clientSet ClientSet.ClientSet) error {
	if codebaseSettings.UserSettings.VcsIntegrationEnabled {
		return gerrit.SetupProjectReplication(codebaseSettings, clientSet)
	}
	log.Print("Skipped gerrit replication configuration. VCS integration isn't enabled")
	return nil
}

func (s CodebaseService) trySetupPerf(codebaseSettings model.CodebaseSettings) error {
	if codebaseSettings.UserSettings.PerfIntegrationEnabled {
		return s.setupPerf(codebaseSettings)
	}
	log.Print("Skipped perf configuration. Perf integration isn't enabled")
	return nil
}

func (s CodebaseService) setupPerf(codebaseSettings model.CodebaseSettings) error {
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

	err = trySetupGitlabPerf(perfClient, codebaseSettings, perfSetting[perf.GitlabDsSettingsKey])
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

func trySetupGitlabPerf(client *perf.Client, codebaseSettings model.CodebaseSettings, dsId string) error {
	if isGitLab(codebaseSettings) {
		return setupGitlabPerf(client, codebaseSettings.VcsProjectPath, dsId)
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

func isGitLab(codebaseSettings model.CodebaseSettings) bool {
	return codebaseSettings.UserSettings.VcsIntegrationEnabled &&
		codebaseSettings.UserSettings.VcsToolName == model.GitLab
}

func createProjectInGerrit(codebaseSettings *model.CodebaseSettings, s *CodebaseService) error {
	projectExist, err := gerrit.CheckProjectExist(codebaseSettings.GerritKeyPath, codebaseSettings.GerritHost,
		codebaseSettings.GerritSettings.SshPort, s.CustomResource.Name)
	if err != nil {
		return err
	}
	if *projectExist {
		return errors.New("couldn't create project in Gerrit. Project already exists")
	}
	err = gerrit.CreateProject(codebaseSettings.GerritKeyPath, codebaseSettings.GerritHost,
		codebaseSettings.GerritSettings.SshPort, s.CustomResource.Name)
	if err != nil {
		return err
	}
	return nil
}

func pushToGerrit(codebaseSettings *model.CodebaseSettings, s *CodebaseService) error {
	err := gerrit.AddRemoteLinkToGerrit(codebaseSettings.WorkDir+"/"+s.CustomResource.Name,
		codebaseSettings.GerritHost, codebaseSettings.GerritSettings.SshPort, s.CustomResource.Name)
	if err != nil {
		return err
	}
	err = gerrit.PushToGerrit(codebaseSettings.WorkDir+"/"+s.CustomResource.Name, codebaseSettings.GerritKeyPath)
	if err != nil {
		return err
	}
	return nil
}

func tryCreateProjectInVcs(codebaseSettings *model.CodebaseSettings, s *CodebaseService, clientSet ClientSet.ClientSet) error {
	if codebaseSettings.UserSettings.VcsIntegrationEnabled {
		err := createProjectInVcs(codebaseSettings, s, clientSet)
		if err != nil {
			return err
		}
	} else {
		log.Println("VCS integration isn't enabled")
		return nil
	}
	return nil
}

func createProjectInVcs(codebaseSettings *model.CodebaseSettings, s *CodebaseService,
	clientSet ClientSet.ClientSet) error {
	VcsCredentialsSecretName := "vcs-autouser-codebase-" + s.CustomResource.Name + "-temp"
	vcsAutoUserLogin, vcsAutoUserPassword, err := settings.GetVcsBasicAuthConfig(clientSet,
		s.CustomResource.Namespace, VcsCredentialsSecretName)

	vcsTool, err := vcs.CreateVCSClient(model.VCSTool(codebaseSettings.UserSettings.VcsToolName),
		codebaseSettings.ProjectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		log.Printf("Unable to create VCS client: %v", err)
		return err
	}

	projectExist, err := vcsTool.CheckProjectExist(codebaseSettings.ProjectVcsGroupPath, codebaseSettings.Name)
	if err != nil {
		return err
	}
	if *projectExist {
		return errors.New("Couldn't copy project to your VCS group. Repository %s is already exists in " +
			s.CustomResource.Name + "" + codebaseSettings.ProjectVcsGroupPath)
	} else {
		_, err = vcsTool.CreateProject(codebaseSettings.ProjectVcsGroupPath, codebaseSettings.Name)
		if err != nil {
			return err
		}
		codebaseSettings.VcsSshUrl, err = vcsTool.GetRepositorySshUrl(codebaseSettings.ProjectVcsGroupPath, codebaseSettings.Name)
	}
	return err
}

func tryCloneRepo(s CodebaseService, codebaseSettings model.CodebaseSettings, repoUrl string,
	repositoryUsername string, repositoryPassword string) error {
	destination := codebaseSettings.WorkDir + "/" + s.CustomResource.Name
	err := git.CloneRepo(repoUrl, repositoryUsername, repositoryPassword, destination)
	if err != nil {
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
