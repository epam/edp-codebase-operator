package impl

import (
	"business-app-handler-controller/models"
	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"business-app-handler-controller/pkg/gerrit"
	"business-app-handler-controller/pkg/git"
	"business-app-handler-controller/pkg/jenkins"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"business-app-handler-controller/pkg/perf"
	"business-app-handler-controller/pkg/settings"
	"business-app-handler-controller/pkg/vcs"
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

type BusinessApplication struct {
	CustomResource *edpv1alpha1.BusinessApplication
	Client         client.Client
	Scheme         *runtime.Scheme
}

func (businessApplication BusinessApplication) Create() {
	if businessApplication.CustomResource.Status.Status != models.StatusInit {
		log.Println("Application status is not init. Skipped")
		return
	}

	log.Println("Create application...")
	log.Printf("Retrieved params: name: %v; strategy: %v; lang: %v; framework: %v; buildTool: %v; route: %v;"+
		" database: %v; repository: %v",
		businessApplication.CustomResource.Name, businessApplication.CustomResource.Spec.Strategy,
		businessApplication.CustomResource.Spec.Lang, businessApplication.CustomResource.Spec.Framework,
		businessApplication.CustomResource.Spec.BuildTool, businessApplication.CustomResource.Spec.Route,
		businessApplication.CustomResource.Spec.Database, businessApplication.CustomResource.Spec.Repository)

	setStatusFields(businessApplication, false, models.StatusInProgress, time.Now())

	err := businessApplication.Client.Update(context.TODO(), businessApplication.CustomResource)
	if err != nil {
		log.Printf("Error has been occurred in status update: %v", err)
		return
	}
	log.Println("Status of CR has been changed to 'in progress'")

	clientSet := ClientSet.CreateOpenshiftClients()
	log.Println("Client set has been created")

	appSettings, err := initAppSettings(businessApplication, clientSet)
	if err != nil {
		log.Printf("Error has been occurred in init app settings: %v", err)
		rollback(businessApplication)
		return
	}
	log.Println("App settings has been retrieved")

	err = gerritConfiguration(appSettings, businessApplication, clientSet)
	if err != nil {
		log.Printf("Error has been occurred in gerrit configuration: %v", err)
		rollback(businessApplication)
		return
	}
	log.Println("Gerrit has been configured")

	jenkinsClient, err := jenkins.Init(appSettings.JenkinsUrl, appSettings.JenkinsUsername, appSettings.JenkinsToken)
	if err != nil {
		log.Println(err)
		rollback(businessApplication)
		return
	}

	err = jenkinsClient.TriggerJobProvisioning(businessApplication.CustomResource.Name,
		businessApplication.CustomResource.Spec.BuildTool)
	if err != nil {
		rollback(businessApplication)
		return
	}
	log.Println("Job provisioning has been triggered")

	err = trySetupPerf(businessApplication, clientSet, *appSettings)

	config, err := gerrit.ConfigInit(*clientSet, *appSettings, businessApplication.CustomResource.Spec)
	err = gerrit.PushConfigs(*config, *appSettings, *clientSet)
	if err != nil {
		log.Println(err)
		rollback(businessApplication)
		return
	}

	log.Println("Pipelines and templates has been pushed to Gerrit")

	envs, err := settings.GetEnvSettings(*clientSet, appSettings.CicdNamespace)
	if err != nil {
		log.Println(err)
		rollback(businessApplication)
		return
	}

	if len(envs) != 0 {
		for _, env := range envs {
			err = ClientSet.PatchBuildConfig(*clientSet, *appSettings, env)
			if err != nil {
				log.Println(err)
				rollback(businessApplication)
				return
			}
		}

		log.Printf("Build config for %v application has been patched", businessApplication.CustomResource.Name)
	}

	setStatusFields(businessApplication, true, models.StatusFinished, time.Now())
}

func initAppSettings(businessApplication BusinessApplication, clientSet *ClientSet.ClientSet) (*models.AppSettings, error) {
	var workDir = fmt.Sprintf("/home/business-app-handler-controller/edp/%v/%v",
		businessApplication.CustomResource.Namespace, businessApplication.CustomResource.Name)
	appSettings := models.AppSettings{}
	appSettings.BasicPatternUrl = "https://github.com/epmd-edp"
	appSettings.Name = businessApplication.CustomResource.Name

	log.Printf("Retrieving user settings from config map...")
	appSettings.CicdNamespace = businessApplication.CustomResource.Namespace
	appSettings.GerritHost = fmt.Sprintf("gerrit.%v", appSettings.CicdNamespace)
	err := settings.CreateWorkdir(workDir)
	if err != nil {
		return nil, err
	}
	appSettings.WorkDir = workDir
	appSettings.GerritKeyPath = fmt.Sprintf("%v/gerrit-private.key", appSettings.WorkDir)

	userSettings, err := settings.GetUserSettingsConfigMap(*clientSet, businessApplication.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	gerritSettings, err := settings.GetGerritSettingsConfigMap(*clientSet, businessApplication.CustomResource.Namespace)
	if err != nil {
		return nil, err
	}
	appSettings.UserSettings = *userSettings
	appSettings.GerritSettings = *gerritSettings
	appSettings.JenkinsToken, appSettings.JenkinsUsername, err = settings.GetJenkinsCreds(*clientSet,
		businessApplication.CustomResource.Namespace)
	appSettings.JenkinsUrl = fmt.Sprintf("http://jenkins.%s:8080", appSettings.CicdNamespace)

	if appSettings.UserSettings.VcsIntegrationEnabled {
		VcsGroupNameUrl, err := url.Parse(appSettings.UserSettings.VcsGroupNameUrl)
		if err != nil {
			log.Print(err)
		}
		appSettings.ProjectVcsHostname = VcsGroupNameUrl.Host
		appSettings.ProjectVcsGroupPath = VcsGroupNameUrl.Path[1:len(VcsGroupNameUrl.Path)]
		appSettings.ProjectVcsHostnameUrl = VcsGroupNameUrl.Scheme + "://" + appSettings.ProjectVcsHostname
		appSettings.VcsProjectPath = appSettings.ProjectVcsGroupPath + "/" + businessApplication.CustomResource.Name
		appSettings.VcsKeyPath = appSettings.WorkDir + "/vcs-private.key"

		appSettings.VcsAutouserSshKey, appSettings.VcsAutouserEmail, err = settings.GetVcsCredentials(*clientSet,
			businessApplication.CustomResource.Namespace)
	} else {
		log.Printf("VCS integration isn't enabled")
	}

	appSettings.GerritPrivateKey, appSettings.GerritPublicKey, err = settings.GetGerritCredentials(*clientSet,
		businessApplication.CustomResource.Namespace)

	log.Printf("Retrieving settings has been finished.")

	return &appSettings, nil
}

func rollback(businessApplication BusinessApplication) {
	setStatusFields(businessApplication, false, models.StatusFailed, time.Now())
}

func gerritConfiguration(appSettings *models.AppSettings, businessApplication BusinessApplication,
	clientSet *ClientSet.ClientSet) error {
	log.Printf("Start gerrit configuration for app: %v...", appSettings.Name)

	log.Printf("Start creation of gerrit private key for app: %v...", appSettings.Name)
	err := settings.CreateGerritPrivateKey(appSettings.GerritPrivateKey, appSettings.GerritKeyPath)
	if err != nil {
		log.Printf("Creation of gerrit private key for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Start creation of ssh config for app: %v...", appSettings.Name)
	err = settings.CreateSshConfig(*appSettings)
	if err != nil {
		log.Printf("Creation of ssh config for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Start setup repo url for app: %v...", appSettings.Name)

	repoUrl, err := getRepoUrl(appSettings.BasicPatternUrl, businessApplication.CustomResource.Spec)

	if err != nil {
		log.Printf("Setup repo url for app %v has been failed. Return error", appSettings.Name)
		return err
	}

	log.Printf("Repository URL to clone sources has been retrieved: %v", *repoUrl)

	repositoryCredentialsSecretName := fmt.Sprintf("repository-application-%v-temp", businessApplication.CustomResource.Name)
	repositoryUsername, repositoryPassword, err := settings.GetVcsBasicAuthConfig(*clientSet,
		businessApplication.CustomResource.Namespace, repositoryCredentialsSecretName)
	if err != nil {
		log.Printf("Unable to get VCS credentials from secret %v", repositoryCredentialsSecretName)
		return err
	}

	isRepositoryAccessible := git.CheckPermissions(*repoUrl, repositoryUsername, repositoryPassword)
	if !isRepositoryAccessible {
		return fmt.Errorf("user %v cannot get access to the repository %v", repositoryUsername, repoUrl)
	}
	log.Printf("Start creation project in VCS for app: %v...", appSettings.Name)
	err = tryCreateProjectInVcs(appSettings, &businessApplication, *clientSet)
	if err != nil {
		log.Printf("Creation project in VCS for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Start clone project for app: %v...", appSettings.Name)
	err = tryCloneRepo(businessApplication, *appSettings, *repoUrl, repositoryUsername, repositoryPassword)
	if err != nil {
		log.Printf("Clone project for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Start creation project in Gerrit for app: %v...", appSettings.Name)
	err = createProjectInGerrit(appSettings, &businessApplication)
	if err != nil {
		log.Printf("Creation project in Gerrit for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Start push project to Gerrit for app: %v...", appSettings.Name)
	err = pushToGerrit(appSettings, &businessApplication)
	if err != nil {
		log.Printf("Push to gerrit for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Start setup Gerrit replication for app: %v...", appSettings.Name)
	err = trySetupGerritReplication(*appSettings, *clientSet)
	if err != nil {
		log.Printf("Setup gerrit replication for app %v has been failed. Return error", appSettings.Name)
		return err
	}
	log.Printf("Gerrit configuration has been finished successfully for app: %v...", appSettings.Name)
	return nil
}

func trySetupGerritReplication(appSettings models.AppSettings, clientSet ClientSet.ClientSet) error {
	if appSettings.UserSettings.VcsIntegrationEnabled {
		return gerrit.SetupProjectReplication(appSettings, clientSet)
	}
	log.Print("Skipped gerrit replication configuration. VCS integration isn't enabled")
	return nil
}

func trySetupPerf(app BusinessApplication, set *ClientSet.ClientSet, appSettings models.AppSettings) error {
	if appSettings.UserSettings.PerfIntegrationEnabled {
		return setupPerf(app, set, appSettings)
	}
	log.Print("Skipped perf configuration. Perf integration isn't enabled")
	return nil
}

func setupPerf(app BusinessApplication, set *ClientSet.ClientSet, appSettings models.AppSettings) error {
	log.Println("Start perf configuration...")
	perfSetting := perf.GetPerfSettings(*set, app.CustomResource.Namespace)
	log.Printf("Perf setting have been retrieved: %v", perfSetting)
	secret := perf.GetPerfCredentials(*set, app.CustomResource.Namespace)

	perfUrl := perfSetting[perf.UrlSettingsKey]
	user := string(secret["username"])
	log.Printf("Username for perf integration has been retrieved: %v", user)
	pass := string(secret["password"])
	perfClient, err := perf.NewRestClient(perfUrl, user, pass)
	if err != nil {
		log.Printf("Error has occurred during perf client init: %v", err)
		return err
	}

	err = setupJenkinsPerf(perfClient, app.CustomResource.Name, perfSetting[perf.JenkinsDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Jenkins Perf: %v", err)
		return err
	}

	err = setupSonarPerf(perfClient, app.CustomResource.Name, perfSetting[perf.SonarDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Sonar Perf: %v", err)
		return err
	}

	err = setupGerritPerf(perfClient, app.CustomResource.Name, perfSetting[perf.GerritDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Gerrit Perf: %v", err)
		return err
	}

	err = trySetupGitlabPerf(perfClient, appSettings, perfSetting[perf.GitlabDsSettingsKey])
	if err != nil {
		log.Printf("Error has occurred during setup Gitlab Perf: %v", err)
		return err
	}

	log.Println("Perf integration has been successfully finished")
	return nil
}

func setupJenkinsPerf(client *perf.Client, appName string, dsId string) error {
	jenkinsDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}
	jenkinsJobs := []string{fmt.Sprintf("/Code-review-%s", appName), fmt.Sprintf("/Build-%s", appName)}
	return client.AddJobsToJenkinsDS(jenkinsDsID, jenkinsJobs)
}

func setupSonarPerf(client *perf.Client, appName string, dsId string) error {
	sonarDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}
	sonarProjects := []string{fmt.Sprintf("%s:master", appName)}
	return client.AddProjectsToSonarDS(sonarDsID, sonarProjects)
}

func setupGerritPerf(client *perf.Client, appName string, dsId string) error {
	gerritDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}

	gerritProjects := []perf.GerritPerfConfig{{ProjectName: appName, Branches: []string{"master"}}}
	return client.AddProjectsToGerritDS(gerritDsID, gerritProjects)
}

func trySetupGitlabPerf(client *perf.Client, appSettings models.AppSettings, dsId string) error {
	if isGitLab(appSettings) {
		return setupGitlabPerf(client, appSettings.VcsProjectPath, dsId)
	}
	return nil
}

func setupGitlabPerf(client *perf.Client, appName string, dsId string) error {
	gitDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}

	gitProjects := map[string]string{appName: "master"}
	return client.AddRepositoriesToGitlabDS(gitDsID, gitProjects)
}

func isGitLab(appSettings models.AppSettings) bool {
	return appSettings.UserSettings.VcsIntegrationEnabled &&
		appSettings.UserSettings.VcsToolName == models.GitLab
}

func createProjectInGerrit(appSettings *models.AppSettings, application *BusinessApplication) error {
	projectExist, err := gerrit.CheckProjectExist(appSettings.GerritKeyPath, appSettings.GerritHost,
		appSettings.GerritSettings.SshPort, application.CustomResource.Name)
	if err != nil {
		return err
	}
	if *projectExist {
		return errors.New("couldn't create project in Gerrit. Project already exists")
	}
	err = gerrit.CreateProject(appSettings.GerritKeyPath, appSettings.GerritHost,
		appSettings.GerritSettings.SshPort, application.CustomResource.Name)
	if err != nil {
		return err
	}
	return nil
}

func pushToGerrit(appSettings *models.AppSettings, businessApplication *BusinessApplication) error {
	err := gerrit.AddRemoteLinkToGerrit(appSettings.WorkDir+"/"+businessApplication.CustomResource.Name,
		appSettings.GerritHost, appSettings.GerritSettings.SshPort, businessApplication.CustomResource.Name)
	if err != nil {
		return err
	}
	err = gerrit.PushToGerrit(appSettings.WorkDir+"/"+businessApplication.CustomResource.Name, appSettings.GerritKeyPath)
	if err != nil {
		return err
	}
	return nil
}

func tryCreateProjectInVcs(appSettings *models.AppSettings, application *BusinessApplication, clientSet ClientSet.ClientSet) error {
	if appSettings.UserSettings.VcsIntegrationEnabled {
		err := createProjectInVcs(appSettings, application, clientSet)
		if err != nil {
			return err
		}
	} else {
		log.Println("VCS integration isn't enabled")
		return nil
	}
	return nil
}

func createProjectInVcs(appSettings *models.AppSettings, application *BusinessApplication,
	clientSet ClientSet.ClientSet) error {
	VcsCredentialsSecretName := "vcs-autouser-application-" + application.CustomResource.Name + "-temp"
	vcsAutoUserLogin, vcsAutoUserPassword, err := settings.GetVcsBasicAuthConfig(clientSet,
		application.CustomResource.Namespace, VcsCredentialsSecretName)

	vcsTool, err := vcs.CreateVCSClient(models.VCSTool(appSettings.UserSettings.VcsToolName),
		appSettings.ProjectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		log.Printf("Unable to create VCS client: %v", err)
		return err
	}

	projectExist, err := vcsTool.CheckProjectExist(appSettings.ProjectVcsGroupPath, appSettings.Name)
	if err != nil {
		return err
	}
	if *projectExist {
		return errors.New("Couldn't copy project to your VCS group. Repository %s is already exists in " +
			application.CustomResource.Name + "" + appSettings.ProjectVcsGroupPath)
	} else {
		_, err = vcsTool.CreateProject(appSettings.ProjectVcsGroupPath, appSettings.Name)
		if err != nil {
			return err
		}
		appSettings.VcsSshUrl, err = vcsTool.GetRepositorySshUrl(appSettings.ProjectVcsGroupPath, appSettings.Name)
	}
	return err
}

func tryCloneRepo(businessApplication BusinessApplication, appSettings models.AppSettings, repoUrl string,
	repositoryUsername string, repositoryPassword string) error {
	destination := appSettings.WorkDir + "/" + businessApplication.CustomResource.Name
	err := git.CloneRepo(repoUrl, repositoryUsername, repositoryPassword, destination)
	if err != nil {
		return err
	}
	log.Printf("Repository has been cloned to %v", destination)
	return nil
}

func setStatusFields(businessApplication BusinessApplication, available bool, status string, time time.Time) {
	businessApplication.CustomResource.Status.Status = status
	businessApplication.CustomResource.Status.LastTimeUpdated = time
	businessApplication.CustomResource.Status.Available = available
	log.Printf("Status for application %v has been updated to '%v' at %v. Available: %v",
		businessApplication.CustomResource.Name, status, time, available)
}

func (businessApplication BusinessApplication) Update() {

}

func (businessApplication BusinessApplication) Delete() {

}
