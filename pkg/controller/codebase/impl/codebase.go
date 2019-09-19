package impl

import (
	"context"
	"errors"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/models"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
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
	"strconv"
	"strings"
	"time"
)

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
)

func (s CodebaseService) Create() error {
	if s.CustomResource.Status.Status != models.StatusInit {
		return errors.New("codebase status is not init. skipped")
	}

	log.Printf("Creating codebase %v ...", s.CustomResource.Spec.Type)
	if s.CustomResource.Spec.Type == Application {
		log.Printf("Retrieved params: name: %v; strategy: %v; lang: %v; framework: %v; buildTool: %v; route: %v;"+
			" database: %v; repository: %v; type: %v; git host: %v; git repo path: %v;",
			s.CustomResource.Name, s.CustomResource.Spec.Strategy, s.CustomResource.Spec.Lang,
			*s.CustomResource.Spec.Framework, s.CustomResource.Spec.BuildTool, s.CustomResource.Spec.Route,
			s.CustomResource.Spec.Database, s.CustomResource.Spec.Repository, s.CustomResource.Spec.Type,
			s.CustomResource.Spec.GitServer, s.CustomResource.Spec.GitUrlPath)
	} else if s.CustomResource.Spec.Type == Autotests {
		log.Printf("Retrieved params: name: %v; strategy: %v; lang: %v; buildTool: %v; route: %v;"+
			" database: %v; repository: %v; type: %v",
			s.CustomResource.Name, s.CustomResource.Spec.Strategy, s.CustomResource.Spec.Lang,
			s.CustomResource.Spec.BuildTool, s.CustomResource.Spec.Route, s.CustomResource.Spec.Database,
			s.CustomResource.Spec.Repository, s.CustomResource.Spec.Type)
	}

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

	var codebaseSettings *models.CodebaseSettings

	if s.CustomResource.Spec.Type == Application && s.CustomResource.Spec.Strategy == ImportStrategy {
		log.Println("Start executing flow for Import strategy")

		codebaseSettings, err = s.initCodebaseSettingsForImportStrategy()
		if err != nil {
			setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
			return errWrap.Wrap(err, "Error has been occurred in init codebase settings for Import strategy")
		}

		log.Println("Codebase settings has been retrieved")

		gitServerRequest, err := s.OpenshiftService.GetGitServer(s.CustomResource.Spec.GitServer, codebaseSettings.CicdNamespace)
		if err != nil {
			return err
		}

		gitServer, err := model.ConvertToGitServer(*gitServerRequest)
		if err != nil {
			return err
		}
		codebaseSettings.GitServer = *gitServer

		err = s.pushBuildConfigs(codebaseSettings)
		if err != nil {
			setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
			return errWrap.Wrap(err, "an error has been occurred while pushing build configs")
		}

		sshLink := s.generateSshLink(*codebaseSettings)
		log.Printf("Repository path: %v", sshLink)

		err = s.triggerJobProvisioning(model.Jenkins{
			JenkinsUrl:      codebaseSettings.JenkinsUrl,
			JenkinsUsername: codebaseSettings.JenkinsUsername,
			JenkinsToken:    codebaseSettings.JenkinsToken,
		},
			map[string]string{
				"PARAM":                 "true",
				"NAME":                  s.CustomResource.Name,
				"BUILD_TOOL":            strings.ToLower(s.CustomResource.Spec.BuildTool),
				"GIT_SERVER":            gitServer.GitHost,
				"GIT_SSH_PORT":          strconv.FormatInt(gitServer.SshPort, 10),
				"GIT_USERNAME":          gitServer.GitUser,
				"GIT_SERVER_CR_NAME":    gitServer.Name,
				"GIT_SERVER_CR_VERSION": "v2",
				"GIT_CREDENTIALS_ID":    gitServer.NameSshKeySecret,
				"REPOSITORY_PATH":       sshLink,
			})
		if err != nil {
			setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
			return errWrap.Wrap(err, "an error has been occurred while triggering job provisioning")
		}

		setIntermediateSuccessFields(s, edpv1alpha1.JenkinsConfiguration)
		log.Println("Job provisioning has been triggered")
	} else {
		log.Println("Start executing flow for Clone or Create strategy")

		codebaseSettings, err = s.initCodebaseSettings(s.ClientSet)
		if err != nil {
			setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
			return errWrap.Wrap(err, "Error has been occurred in init codebase settings")
		}

		err = s.gerritConfiguration(codebaseSettings)
		if err != nil {
			setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
			return errWrap.Wrap(err, "Error has been occurred in gerrit configuration")
		}

		log.Println("Codebase settings has been retrieved")

		setIntermediateSuccessFields(s, edpv1alpha1.GerritRepositoryProvisioning)

		log.Println("Gerrit has been configured")

		gitServerRequest, err := s.OpenshiftService.GetGitServer(GerritGitServerName, codebaseSettings.CicdNamespace)
		if err != nil {
			return err
		}

		gitServer, err := model.ConvertToGitServer(*gitServerRequest)
		if err != nil {
			return err
		}
		codebaseSettings.GitServer = *gitServer

		sshLink := s.generateSshLink(*codebaseSettings)
		log.Printf("Repository path: %v", sshLink)

		err = s.triggerJobProvisioning(model.Jenkins{
			JenkinsUrl:      codebaseSettings.JenkinsUrl,
			JenkinsUsername: codebaseSettings.JenkinsUsername,
			JenkinsToken:    codebaseSettings.JenkinsToken,
		},
			map[string]string{
				"PARAM":                 "true",
				"NAME":                  s.CustomResource.Name,
				"BUILD_TOOL":            strings.ToLower(s.CustomResource.Spec.BuildTool),
				"GIT_SERVER":            gitServer.GitHost,
				"GIT_SSH_PORT":          strconv.FormatInt(gitServer.SshPort, 10),
				"GIT_USERNAME":          gitServer.GitUser,
				"GIT_SERVER_CR_NAME":    GerritGitServerName,
				"GIT_SERVER_CR_VERSION": "v2",
				"GIT_CREDENTIALS_ID":    gitServer.NameSshKeySecret,
				"REPOSITORY_PATH":       sshLink,
			})
		if err != nil {
			setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
			return errWrap.Wrap(err, "an error has been occurred while triggering job provisioning")
		}

		setIntermediateSuccessFields(s, edpv1alpha1.JenkinsConfiguration)
		log.Println("Job provisioning has been triggered")

		err = s.trySetupPerf(*codebaseSettings)

		config, err := gerrit.ConfigInit(*s.ClientSet, *codebaseSettings, s.CustomResource.Spec)
		err = gerrit.PushConfigs(*config, *codebaseSettings, *s.ClientSet)
		if err != nil {
			setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
			return err
		}

		log.Println("Pipelines and templates has been pushed to Gerrit")
	}

	err = os.RemoveAll(codebaseSettings.WorkDir)
	if err != nil {
		return err
	}
	log.Printf("Workdir %v has been cleaned", codebaseSettings.WorkDir)

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

func (s CodebaseService) generateSshLink(codebaseSettings models.CodebaseSettings) string {
	return fmt.Sprintf("ssh://%v@%v:%v%v", codebaseSettings.GitServer.GitUser, codebaseSettings.GitServer.GitHost,
		codebaseSettings.GitServer.SshPort, codebaseSettings.RepositoryPath)
}

func (s CodebaseService) pushBuildConfigs(codebaseSettings *models.CodebaseSettings) error {
	log.Println("Start pushing build configs to remote git server...")

	gitServer := codebaseSettings.GitServer

	secret, err := s.OpenshiftService.GetSecret(gitServer.NameSshKeySecret, codebaseSettings.CicdNamespace)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred  while getting %v secret", gitServer.NameSshKeySecret))
	}

	templatesDir := fmt.Sprintf("%v/oc-templates", codebaseSettings.WorkDir)
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

	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", fmt.Sprintf("%v/oc-templates", codebaseSettings.WorkDir),
		codebaseSettings.Name)
	err = util.CreateDirectory(appTemplatesDir)
	if err != nil {
		return errWrap.Wrap(err, fmt.Sprintf("an error has occurred while creating template folder %v", templatesDir))
	}

	templateBasePath := fmt.Sprintf("/usr/local/bin/templates/applications/%v", strings.ToLower(codebaseSettings.Lang))
	templateName := fmt.Sprintf("%v.tmpl", strings.ToLower(codebaseSettings.Framework))
	templatePath := fmt.Sprintf("%v/%v", templateBasePath, templateName)
	templateConfig := buildTemplateConfig(*codebaseSettings, s.CustomResource.Spec)

	err = util.CopyTemplate(templatePath, templateName, templateConfig)
	if err != nil {
		return errWrap.Wrap(err, "an error has occurred while copying template")
	}

	err = util.CopyPipelines(PipelinesSourcePath, pathToCopiedGitFolder)
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

	appImageStream, err := gerrit.GetAppImageStream(codebaseSettings.Lang)
	if err != nil {
		return err
	}

	err = gerrit.CreateS2IImageStream(*s.ClientSet, codebaseSettings.Name, codebaseSettings.CicdNamespace, appImageStream)
	if err != nil {
		return err
	}

	log.Println("End pushing build configs to remote git server...")

	return nil
}

func buildTemplateConfig(codebaseSettings models.CodebaseSettings, spec v1alpha1.CodebaseSpec) gerrit.GerritConfigGoTemplating {
	log.Print("Start configuring template config ...")

	config := gerrit.GerritConfigGoTemplating{
		Lang:             spec.Lang,
		Framework:        spec.Framework,
		BuildTool:        spec.BuildTool,
		CodebaseSettings: codebaseSettings,
	}
	if spec.Repository != nil {
		config.RepositoryUrl = &spec.Repository.Url
	}
	if spec.Database != nil {
		config.Database = spec.Database
	}
	if spec.Route != nil {
		config.Route = spec.Route
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

	err = jenkinsClient.TriggerJobProvisioning(parameters, 10*time.Second, 12)
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

func (s CodebaseService) initCodebaseSettingsForImportStrategy() (*models.CodebaseSettings, error) {
	var (
		err     error
		workDir = fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace, s.CustomResource.Name)
	)

	err = settings.CreateWorkdir(workDir)
	if err != nil {
		return nil, err
	}
	codebaseSettings := models.CodebaseSettings{}
	codebaseSettings.WorkDir = workDir
	codebaseSettings.Lang = s.CustomResource.Spec.Lang
	codebaseSettings.Name = s.CustomResource.Name
	codebaseSettings.Type = s.CustomResource.Spec.Type
	codebaseSettings.CicdNamespace = s.CustomResource.Namespace
	codebaseSettings.Framework = *s.CustomResource.Spec.Framework
	codebaseSettings.RepositoryPath = *s.CustomResource.Spec.GitUrlPath

	codebaseSettings.JenkinsToken, codebaseSettings.JenkinsUsername, err = settings.GetJenkinsCreds(*s.ClientSet,
		s.CustomResource.Namespace)
	codebaseSettings.JenkinsUrl = fmt.Sprintf("http://jenkins.%s:8080", codebaseSettings.CicdNamespace)

	if codebaseSettings.UserSettings.VcsIntegrationEnabled {
		VcsGroupNameUrl, err := url.Parse(codebaseSettings.UserSettings.VcsGroupNameUrl)
		if err != nil {
			log.Print(err)
		}
		codebaseSettings.ProjectVcsHostname = VcsGroupNameUrl.Host
		codebaseSettings.ProjectVcsGroupPath = VcsGroupNameUrl.Path[1:len(VcsGroupNameUrl.Path)]
		codebaseSettings.ProjectVcsHostnameUrl = VcsGroupNameUrl.Scheme + "://" + codebaseSettings.ProjectVcsHostname
		codebaseSettings.VcsProjectPath = codebaseSettings.ProjectVcsGroupPath + "/" + s.CustomResource.Name
		codebaseSettings.VcsKeyPath = codebaseSettings.WorkDir + "/vcs-private.key"

		codebaseSettings.VcsAutouserSshKey, codebaseSettings.VcsAutouserEmail, err = settings.GetVcsCredentials(*s.ClientSet,
			s.CustomResource.Namespace)
	} else {
		log.Printf("VCS integration isn't enabled")
	}

	log.Printf("Retrieving settings has been finished.")

	return &codebaseSettings, err
}

func (s CodebaseService) initCodebaseSettings(clientSet *ClientSet.ClientSet) (*models.CodebaseSettings, error) {
	var workDir = fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace,
		s.CustomResource.Name)
	codebaseSettings := models.CodebaseSettings{}
	codebaseSettings.BasicPatternUrl = "https://github.com/epmd-edp"
	codebaseSettings.Name = s.CustomResource.Name
	codebaseSettings.Type = s.CustomResource.Spec.Type
	codebaseSettings.RepositoryPath = "/" + s.CustomResource.Name

	log.Printf("Retrieving user settings from config map...")
	codebaseSettings.CicdNamespace = s.CustomResource.Namespace
	codebaseSettings.GerritHost = fmt.Sprintf("gerrit.%v", codebaseSettings.CicdNamespace)
	err := settings.CreateWorkdir(workDir)
	if err != nil {
		return nil, err
	}
	codebaseSettings.WorkDir = workDir
	codebaseSettings.GerritKeyPath = fmt.Sprintf("%v/gerrit-private.key", codebaseSettings.WorkDir)

	userSettings, err := settings.GetUserSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}

	gerritSettings, err := settings.GetGerritSettingsConfigMap(*s.ClientSet, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}

	codebaseSettings.UserSettings = *userSettings
	codebaseSettings.GerritSettings = *gerritSettings
	codebaseSettings.JenkinsToken, codebaseSettings.JenkinsUsername, err = settings.GetJenkinsCreds(*s.ClientSet,
		s.CustomResource.Namespace)
	codebaseSettings.JenkinsUrl = fmt.Sprintf("http://jenkins.%s:8080", codebaseSettings.CicdNamespace)

	if codebaseSettings.UserSettings.VcsIntegrationEnabled {
		VcsGroupNameUrl, err := url.Parse(codebaseSettings.UserSettings.VcsGroupNameUrl)
		if err != nil {
			log.Print(err)
		}
		codebaseSettings.ProjectVcsHostname = VcsGroupNameUrl.Host
		codebaseSettings.ProjectVcsGroupPath = VcsGroupNameUrl.Path[1:len(VcsGroupNameUrl.Path)]
		codebaseSettings.ProjectVcsHostnameUrl = VcsGroupNameUrl.Scheme + "://" + codebaseSettings.ProjectVcsHostname
		codebaseSettings.VcsProjectPath = codebaseSettings.ProjectVcsGroupPath + "/" + s.CustomResource.Name
		codebaseSettings.VcsKeyPath = codebaseSettings.WorkDir + "/vcs-private.key"

		codebaseSettings.VcsAutouserSshKey, codebaseSettings.VcsAutouserEmail, err = settings.GetVcsCredentials(*s.ClientSet,
			s.CustomResource.Namespace)
	} else {
		log.Printf("VCS integration isn't enabled")
	}

	codebaseSettings.GerritPrivateKey, codebaseSettings.GerritPublicKey, err = settings.GetGerritCredentials(*s.ClientSet,
		s.CustomResource.Namespace)

	log.Printf("Retrieving settings has been finished.")

	return &codebaseSettings, nil
}

func (s CodebaseService) gerritConfiguration(codebaseSettings *models.CodebaseSettings) error {
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

func trySetupGerritReplication(codebaseSettings models.CodebaseSettings, clientSet ClientSet.ClientSet) error {
	if codebaseSettings.UserSettings.VcsIntegrationEnabled {
		return gerrit.SetupProjectReplication(codebaseSettings, clientSet)
	}
	log.Print("Skipped gerrit replication configuration. VCS integration isn't enabled")
	return nil
}

func (s CodebaseService) trySetupPerf(codebaseSettings models.CodebaseSettings) error {
	if codebaseSettings.UserSettings.PerfIntegrationEnabled {
		return s.setupPerf(codebaseSettings)
	}
	log.Print("Skipped perf configuration. Perf integration isn't enabled")
	return nil
}

func (s CodebaseService) setupPerf(codebaseSettings models.CodebaseSettings) error {
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

func trySetupGitlabPerf(client *perf.Client, codebaseSettings models.CodebaseSettings, dsId string) error {
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

func isGitLab(codebaseSettings models.CodebaseSettings) bool {
	return codebaseSettings.UserSettings.VcsIntegrationEnabled &&
		codebaseSettings.UserSettings.VcsToolName == models.GitLab
}

func createProjectInGerrit(codebaseSettings *models.CodebaseSettings, s *CodebaseService) error {
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

func pushToGerrit(codebaseSettings *models.CodebaseSettings, s *CodebaseService) error {
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

func tryCreateProjectInVcs(codebaseSettings *models.CodebaseSettings, s *CodebaseService, clientSet ClientSet.ClientSet) error {
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

func createProjectInVcs(codebaseSettings *models.CodebaseSettings, s *CodebaseService,
	clientSet ClientSet.ClientSet) error {
	VcsCredentialsSecretName := "vcs-autouser-codebase-" + s.CustomResource.Name + "-temp"
	vcsAutoUserLogin, vcsAutoUserPassword, err := settings.GetVcsBasicAuthConfig(clientSet,
		s.CustomResource.Namespace, VcsCredentialsSecretName)

	vcsTool, err := vcs.CreateVCSClient(models.VCSTool(codebaseSettings.UserSettings.VcsToolName),
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

func tryCloneRepo(s CodebaseService, codebaseSettings models.CodebaseSettings, repoUrl string,
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
