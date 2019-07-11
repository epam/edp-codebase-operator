package impl

import (
	"codebase-operator/models"
	edpv1alpha1 "codebase-operator/pkg/apis/edp/v1alpha1"
	"codebase-operator/pkg/gerrit"
	"codebase-operator/pkg/git"
	"codebase-operator/pkg/jenkins"
	ClientSet "codebase-operator/pkg/openshift"
	"codebase-operator/pkg/perf"
	"codebase-operator/pkg/settings"
	"codebase-operator/pkg/vcs"
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"net/url"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

type CodebaseService struct {
	CustomResource *edpv1alpha1.Codebase
	Client         client.Client
	Scheme         *runtime.Scheme
}

func (s CodebaseService) Create() {
	if s.CustomResource.Status.Status != models.StatusInit {
		log.Println("Codebase status is not init. Skipped")
		return
	}

	log.Printf("Creating codebase %v ...", s.CustomResource.Spec.Type)
	if s.CustomResource.Spec.Type == "application" {
		log.Printf("Retrieved params: name: %v; strategy: %v; lang: %v; framework: %v; buildTool: %v; route: %v;"+
			" database: %v; repository: %v; type: %v",
			s.CustomResource.Name, s.CustomResource.Spec.Strategy, s.CustomResource.Spec.Lang,
			*s.CustomResource.Spec.Framework, s.CustomResource.Spec.BuildTool, s.CustomResource.Spec.Route,
			s.CustomResource.Spec.Database, s.CustomResource.Spec.Repository, s.CustomResource.Spec.Type)
	} else if s.CustomResource.Spec.Type == "autotests" {
		log.Printf("Retrieved params: name: %v; strategy: %v; lang: %v; buildTool: %v; route: %v;"+
			" database: %v; repository: %v; type: %v",
			s.CustomResource.Name, s.CustomResource.Spec.Strategy, s.CustomResource.Spec.Lang,
			s.CustomResource.Spec.BuildTool, s.CustomResource.Spec.Route, s.CustomResource.Spec.Database,
			s.CustomResource.Spec.Repository, s.CustomResource.Spec.Type)
	}

	statusCR := edpv1alpha1.CodebaseStatus{
		Available:       false,
		LastTimeUpdated: time.Now(),
		Action:          edpv1alpha1.AcceptCodebaseRegistration,
		Result:          edpv1alpha1.Success,
		Username:        "system",
		Value:           "inactive",
	}

	err := updateStatusFields(s, statusCR)
	if err != nil {
		log.Printf("Error has been occurred in status update: %v", err)
		return
	}
	log.Println("Status of codebase CR has been changed to 'in progress'")

	clientSet := ClientSet.CreateOpenshiftClients()
	log.Println("Client set has been created")

	codebaseSettings, err := initCodebaseSettings(s, clientSet)
	if err != nil {
		log.Printf("Error has been occurred in init codebase settings: %v", err)
		setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return
	}
	log.Println("Codebase settings has been retrieved")

	err = gerritConfiguration(codebaseSettings, s, clientSet)
	if err != nil {
		log.Printf("Error has been occurred in gerrit configuration: %v", err)
		setFailedFields(s, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return
	}
	setIntermediateSuccessFields(s, edpv1alpha1.GerritRepositoryProvisioning)
	log.Println("Gerrit has been configured")

	jenkinsClient, err := jenkins.Init(codebaseSettings.JenkinsUrl, codebaseSettings.JenkinsUsername,
		codebaseSettings.JenkinsToken)
	if err != nil {
		log.Println(err)
		setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
		return
	}
	err = jenkinsClient.TriggerJobProvisioning(s.CustomResource.Name, s.CustomResource.Spec.BuildTool)
	if err != nil {
		log.Println(err)
		setFailedFields(s, edpv1alpha1.JenkinsConfiguration, err.Error())
		return
	}
	setIntermediateSuccessFields(s, edpv1alpha1.JenkinsConfiguration)
	log.Println("Job provisioning has been triggered")

	err = trySetupPerf(s, clientSet, *codebaseSettings)

	config, err := gerrit.ConfigInit(*clientSet, *codebaseSettings, s.CustomResource.Spec)
	err = gerrit.PushConfigs(*config, *codebaseSettings, *clientSet)
	if err != nil {
		log.Println(err)
		setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return
	}

	log.Println("Pipelines and templates has been pushed to Gerrit")

	err = tryPatchBuildConfig(s.CustomResource.Spec.Type, s.CustomResource.Name, clientSet, *codebaseSettings)
	if err != nil {
		log.Println(err)
		setFailedFields(s, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return
	}

	err = os.RemoveAll(codebaseSettings.WorkDir)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Workdir %v has been cleaned", codebaseSettings.WorkDir)

	s.CustomResource.Status = edpv1alpha1.CodebaseStatus{
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupDeploymentTemplates,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}

}

func initCodebaseSettings(s CodebaseService, clientSet *ClientSet.ClientSet) (*models.CodebaseSettings, error) {
	var workDir = fmt.Sprintf("/home/codebase-operator/edp/%v/%v", s.CustomResource.Namespace,
		s.CustomResource.Name)
	codebaseSettings := models.CodebaseSettings{}
	codebaseSettings.BasicPatternUrl = "https://github.com/epmd-edp"
	codebaseSettings.Name = s.CustomResource.Name
	codebaseSettings.Type = s.CustomResource.Spec.Type

	log.Printf("Retrieving user settings from config map...")
	codebaseSettings.CicdNamespace = s.CustomResource.Namespace
	codebaseSettings.GerritHost = fmt.Sprintf("gerrit.%v", codebaseSettings.CicdNamespace)
	err := settings.CreateWorkdir(workDir)
	if err != nil {
		return nil, err
	}
	codebaseSettings.WorkDir = workDir
	codebaseSettings.GerritKeyPath = fmt.Sprintf("%v/gerrit-private.key", codebaseSettings.WorkDir)

	userSettings, err := settings.GetUserSettingsConfigMap(*clientSet, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}

	gerritSettings, err := settings.GetGerritSettingsConfigMap(*clientSet, s.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}

	codebaseSettings.UserSettings = *userSettings
	codebaseSettings.GerritSettings = *gerritSettings
	codebaseSettings.JenkinsToken, codebaseSettings.JenkinsUsername, err = settings.GetJenkinsCreds(*clientSet,
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

		codebaseSettings.VcsAutouserSshKey, codebaseSettings.VcsAutouserEmail, err = settings.GetVcsCredentials(*clientSet,
			s.CustomResource.Namespace)
	} else {
		log.Printf("VCS integration isn't enabled")
	}

	codebaseSettings.GerritPrivateKey, codebaseSettings.GerritPublicKey, err = settings.GetGerritCredentials(*clientSet,
		s.CustomResource.Namespace)

	log.Printf("Retrieving settings has been finished.")

	return &codebaseSettings, nil
}

func gerritConfiguration(codebaseSettings *models.CodebaseSettings, s CodebaseService,
	clientSet *ClientSet.ClientSet) error {
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

	repositoryUsername, repositoryPassword, err := tryGetRepositoryCredentials(s, clientSet)

	isRepositoryAccessible := git.CheckPermissions(*repoUrl, repositoryUsername, repositoryPassword)
	if !isRepositoryAccessible {
		return fmt.Errorf("user %v cannot get access to the repository %v", repositoryUsername, *repoUrl)
	}
	log.Printf("Start creation project in VCS for codebase: %v...", codebaseSettings.Name)
	err = tryCreateProjectInVcs(codebaseSettings, &s, *clientSet)
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
	err = trySetupGerritReplication(*codebaseSettings, *clientSet)
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

func tryPatchBuildConfig(codebaseType string, codebaseName string, clientSet *ClientSet.ClientSet, codebaseSettings models.CodebaseSettings) error {
	if codebaseType == "application" {
		return patchBuildConfig(codebaseName, clientSet, codebaseSettings)
	}

	log.Printf("Codebase type is %v. BuildConfig updating does not required", codebaseType)

	return nil
}

func patchBuildConfig(codebaseName string, clientSet *ClientSet.ClientSet, codebaseSettings models.CodebaseSettings) error {
	envs, err := settings.GetEnvSettings(*clientSet, codebaseSettings.CicdNamespace)
	if err != nil {
		return err
	}

	for _, env := range envs {
		err = ClientSet.PatchBuildConfig(*clientSet, codebaseSettings, env)
		if err != nil {
			return err
		}
	}

	log.Printf("Build config for %v codebase has been patched", codebaseName)

	return nil
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

func trySetupPerf(s CodebaseService, set *ClientSet.ClientSet, codebaseSettings models.CodebaseSettings) error {
	if codebaseSettings.UserSettings.PerfIntegrationEnabled {
		return setupPerf(s, set, codebaseSettings)
	}
	log.Print("Skipped perf configuration. Perf integration isn't enabled")
	return nil
}

func setupPerf(s CodebaseService, set *ClientSet.ClientSet, codebaseSettings models.CodebaseSettings) error {
	log.Println("Start perf configuration...")
	perfSetting := perf.GetPerfSettings(*set, s.CustomResource.Namespace)
	log.Printf("Perf setting have been retrieved: %v", perfSetting)
	secret := perf.GetPerfCredentials(*set, s.CustomResource.Namespace)

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
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          edpv1alpha1.Success,
		Value:           "inactive",
	}
	err := s.Client.Update(context.TODO(), s.CustomResource)
	if err != nil {
		log.Printf("Error has been occured during the update success fields fot codebase: %v", s.CustomResource.Name)
	}
}

func updateStatusFields(service CodebaseService, status edpv1alpha1.CodebaseStatus) error {
	service.CustomResource.Status = status
	return service.Client.Update(context.TODO(), service.CustomResource)
}

func (s CodebaseService) Update() {

}

func (s CodebaseService) Delete() {

}
