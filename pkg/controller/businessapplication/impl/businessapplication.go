package impl

import (
	"business-app-handler-controller/models"
	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"business-app-handler-controller/pkg/git"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"business-app-handler-controller/pkg/perf"
	"business-app-handler-controller/pkg/settings"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type BusinessApplication struct {
	CustomResource *edpv1alpha1.BusinessApplication
	Client         client.Client
	Scheme         *runtime.Scheme
}

func (businessApplication BusinessApplication) Create(allowedAppSettings map[string][]string) {
	appSettings := models.AppSettings{}
	appSettings.BasicPatternUrl = "https://github.com/epmd-edp"
	clientSet := ClientSet.CreateOpenshiftClients()

	log.Printf("Retrieving user settings from config map...")
	appSettings.CicdNamespace = businessApplication.CustomResource.Namespace
	appSettings.WorkDir = "/home/edp"
	appSettings.GerritKeyPath = appSettings.WorkDir + "/gerrit-private.key"

	appSettings.UserSettings = settings.GetUserSettingsConfigMap(*clientSet, businessApplication.CustomResource.Namespace)
	appSettings.GerritSettings = settings.GetGerritSettingsConfigMap(*clientSet, businessApplication.CustomResource.Namespace)
	appSettings.JenkinsToken, appSettings.JenkinsUsername = settings.GetJenkinsCreds(*clientSet, businessApplication.CustomResource.Namespace)

	if appSettings.UserSettings.VcsIntegrationEnabled {
		VcsGroupNameUrl, err := url.Parse(appSettings.UserSettings.VcsGroupNameUrl)
		if err != nil {
			log.Fatal(err)
		}
		appSettings.ProjectVcsHostname = VcsGroupNameUrl.Host
		appSettings.ProjectVcsGroupPath = VcsGroupNameUrl.Path[1:len(VcsGroupNameUrl.Path)]
		appSettings.ProjectVcsHostnameUrl = VcsGroupNameUrl.Scheme + "://" + appSettings.ProjectVcsHostname
		appSettings.VcsProjectPath = appSettings.ProjectVcsGroupPath + "/" + businessApplication.CustomResource.Name
		appSettings.VcsKeyPath = appSettings.WorkDir + "/vcs-private.key"

		appSettings.VcsAutouserSshKey, appSettings.VcsAutouserEmail = settings.GetVcsCredentials(*clientSet, businessApplication.CustomResource.Namespace)
	} else {
		log.Printf("VCS integration isn't enabled")
	}

	appSettings.GerritPrivateKey, appSettings.GerritPublicKey = settings.GetGerritCredentials(*clientSet, businessApplication.CustomResource.Namespace)

	log.Printf("Retrieving settings has been finished.")

	settings.CreateGerritPrivateKey(appSettings.GerritPrivateKey, appSettings.GerritKeyPath)
	settings.CreateSshConfig(appSettings)

	switch strings.ToLower(businessApplication.CustomResource.Spec.Strategy) {
	case strings.ToLower(allowedAppSettings["add_repo_strategy"][0]):
		switch settings.IsFrameworkMultiModule(businessApplication.CustomResource.Spec.Framework) {
		case false:
			//springboot
			if businessApplication.CustomResource.Spec.Database != nil {
				//with database
				appSettings.RepositoryUrl = appSettings.BasicPatternUrl + "/" +
					strings.ToLower(businessApplication.CustomResource.Spec.Lang) +
					"-" + strings.ToLower(businessApplication.CustomResource.Spec.BuildTool) + "-" +
					strings.ToLower(businessApplication.CustomResource.Spec.Framework) + "-" +
					strings.ToLower(businessApplication.CustomResource.Spec.Database.Kind) + ".git"
			} else {
				//without database
				appSettings.RepositoryUrl = appSettings.BasicPatternUrl + "/" +
					strings.ToLower(businessApplication.CustomResource.Spec.Lang) +
					"-" + strings.ToLower(businessApplication.CustomResource.Spec.BuildTool) + "-" +
					strings.ToLower(businessApplication.CustomResource.Spec.Framework) + ".git"
			}
		case true:
			//multi springboot
			if businessApplication.CustomResource.Spec.Database != nil {
				//with database
				appSettings.RepositoryUrl = appSettings.BasicPatternUrl + "/" +
					strings.ToLower(businessApplication.CustomResource.Spec.Lang) +
					"-" + strings.ToLower(businessApplication.CustomResource.Spec.BuildTool) + "-" +
					settings.AddFrameworkMultiModulePostfix(businessApplication.CustomResource.Spec.Framework) + "-" +
					strings.ToLower(businessApplication.CustomResource.Spec.Database.Kind) + ".git"
			} else {
				//without database
				appSettings.RepositoryUrl = appSettings.BasicPatternUrl + "/" + strings.ToLower(businessApplication.CustomResource.Spec.Lang) +
					"-" + strings.ToLower(businessApplication.CustomResource.Spec.BuildTool) + "-" +
					settings.AddFrameworkMultiModulePostfix(businessApplication.CustomResource.Spec.Framework) + ".git"
			}
		default:
			log.Fatalf("Provided unsupported framework - " + businessApplication.CustomResource.Spec.Framework)
		}
	case allowedAppSettings["add_repo_strategy"][1]:
		appSettings.RepositoryUrl = businessApplication.CustomResource.Spec.Git.Url
	default:
		log.Fatalf("Provided unsupported add repository strategy - " + businessApplication.CustomResource.Spec.Strategy)
	}

	VcsCredentialsSecretName := "repository-application-" + businessApplication.CustomResource.Name + "-temp"
	repositoryUsername, repositoryPassword := settings.GetVcsBasicAuthConfig(*clientSet, businessApplication.CustomResource.Namespace, VcsCredentialsSecretName)

	repositoryAccess := git.CheckPermissions(appSettings.RepositoryUrl, repositoryUsername, repositoryPassword)

	if !repositoryAccess {
		log.Fatalf("Cannot access provided git repository: %s", businessApplication.CustomResource.Spec.Git)
	}
	if appSettings.UserSettings.VcsIntegrationEnabled {
		//Implementation for VCS project creation
	}

	err := trySetupPerf(businessApplication, clientSet, appSettings)

	if err != nil {
		rollback()
	}
}

func rollback() {
	//TODO add logic
}

func trySetupPerf(app BusinessApplication, set *ClientSet.OpenshiftClientSet, appSettings models.AppSettings) error {
	if appSettings.UserSettings.PerfIntegrationEnabled {
		return setupPerf(app, set, appSettings)
	} else {
		log.Printf("Perf integration isn't enabled")
	}
	return nil
}

func setupPerf(app BusinessApplication, set *ClientSet.OpenshiftClientSet, appSettings models.AppSettings) error {
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

func (businessApplication BusinessApplication) Update() {

}

func (businessApplication BusinessApplication) Delete() {

}
