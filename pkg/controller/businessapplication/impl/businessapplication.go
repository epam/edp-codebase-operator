package impl

import (
	"business-app-handler-controller/models"
	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"business-app-handler-controller/pkg/git"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"business-app-handler-controller/pkg/perf"
	"business-app-handler-controller/pkg/settings"
	"business-app-handler-controller/pkg/vcs"
	"context"
	"errors"
	"fmt"
	"github.com/bndr/gojenkins"
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

func (businessApplication BusinessApplication) Create() {
	if businessApplication.CustomResource.Status.Status != models.StatusInit {
		return
	}
	businessApplication.CustomResource.Status.Status = models.StatusInProgress
	_ = businessApplication.Client.Update(context.TODO(), businessApplication.CustomResource)

	appSettings := models.AppSettings{}
	appSettings.BasicPatternUrl = "https://github.com/epmd-edp"
	clientSet := ClientSet.CreateOpenshiftClients()

	log.Printf("Retrieving user settings from config map...")
	appSettings.CicdNamespace = businessApplication.CustomResource.Namespace
	appSettings.WorkDir = "/home/edp"
	appSettings.GerritKeyPath = appSettings.WorkDir + "/gerrit-private.key"

	userSettings, err := settings.GetUserSettingsConfigMap(*clientSet, businessApplication.CustomResource.Namespace)
	gerritSettings, err := settings.GetGerritSettingsConfigMap(*clientSet, businessApplication.CustomResource.Namespace)

	appSettings.UserSettings = *userSettings
	appSettings.GerritSettings = *gerritSettings
	appSettings.JenkinsToken, appSettings.JenkinsUsername, err = settings.GetJenkinsCreds(*clientSet,
		businessApplication.CustomResource.Namespace)

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

	_ = settings.CreateGerritPrivateKey(appSettings.GerritPrivateKey, appSettings.GerritKeyPath)
	err = settings.CreateSshConfig(appSettings)

	err = setRepositoryUrl(&appSettings, &businessApplication)

	repositoryCredentialsSecretName := "repository-application-" + businessApplication.CustomResource.Name + "-temp"
	repositoryUsername, repositoryPassword, err := settings.GetVcsBasicAuthConfig(*clientSet,
		businessApplication.CustomResource.Namespace, repositoryCredentialsSecretName)

	isRepositoryAccessible := git.CheckPermissions(appSettings.RepositoryUrl, repositoryUsername, repositoryPassword)
	if isRepositoryAccessible {
		err = tryCreateProjectInVcs(&appSettings, &businessApplication, *clientSet)
	} else {
		log.Printf("Cannot access provided git repository: %s", businessApplication.CustomResource.Spec.Repository.Url)
	}

	err = triggerJobProvisioning(businessApplication, appSettings)
	if err != nil {
		rollback(businessApplication)
		return
	}

	err = trySetupPerf(businessApplication, clientSet, appSettings)

	if err != nil {
		rollback(businessApplication)
		return
	}

	businessApplication.CustomResource.Status.Available = true
	businessApplication.CustomResource.Status.Status = models.StatusFinished
}

func rollback(businessApplication BusinessApplication) {
	businessApplication.CustomResource.Status.Status = models.StatusFailed
}

func triggerJobProvisioning(app BusinessApplication, appSettings models.AppSettings) error {
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", appSettings.CicdNamespace)
	jenkins := gojenkins.CreateJenkins(jenkinsUrl, appSettings.JenkinsUsername, appSettings.JenkinsToken)

	_, err := jenkins.Init()
	if err != nil {
		return err
	}

	_, err = jenkins.BuildJob("Job-provisioning", map[string]string{
		"PARAM":      "true",
		"NAME":       app.CustomResource.Name,
		"TYPE":       "app",
		"BUILD_TOOL": app.CustomResource.Spec.BuildTool,
	})
	return err
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

func tryCreateProjectInVcs(appSettings *models.AppSettings, application *BusinessApplication, clientSet ClientSet.OpenshiftClientSet) error {
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
	clientSet ClientSet.OpenshiftClientSet) error {
	VcsCredentialsSecretName := "vcs-autouser-application-" + application.CustomResource.Name + "-temp"
	vcsAutoUserLogin, vcsAutoUserPassword, err := settings.GetVcsBasicAuthConfig(clientSet,
		application.CustomResource.Namespace, VcsCredentialsSecretName)

	vcsTool, err := vcs.CreateVCSClient(models.VCSTool(appSettings.UserSettings.VcsToolName),
		appSettings.ProjectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		log.Printf("Unable to create VCS client: %v", err)
		return err
	}

	projectExist, err := vcsTool.CheckProjectExist(appSettings.VcsProjectPath)
	if err != nil {
		return err
	}
	if projectExist {
		return errors.New("Couldn't copy project to your VCS group. Repository %s is already exists in " +
			application.CustomResource.Name + appSettings.ProjectVcsGroupPath)
	} else {
		groupId, err := vcsTool.GetGroupIdByName(appSettings.ProjectVcsGroupPath)
		if err != nil {
			return err
		}
		err = vcsTool.CreateProject(application.CustomResource.Name, groupId)
		if err != nil {
			return err
		}
	}
	return nil
}

func setRepositoryUrl(appSettings *models.AppSettings, application *BusinessApplication) error {
	switch application.CustomResource.Spec.Strategy {
	case models.Create:
		concatCreateRepoUrl(appSettings, application)
	case models.Clone:
		appSettings.RepositoryUrl = application.CustomResource.Spec.Repository.Url
	}
	return nil
}

func concatCreateRepoUrl(appSettings *models.AppSettings, application *BusinessApplication) {
	repoUrl := appSettings.BasicPatternUrl + "/" +
		strings.ToLower(application.CustomResource.Spec.Lang) + "-" +
		strings.ToLower(application.CustomResource.Spec.BuildTool) + "-" +
		strings.ToLower(application.CustomResource.Spec.Framework)
	if application.CustomResource.Spec.Database != nil {
		appSettings.RepositoryUrl += repoUrl + "-" + strings.ToLower(application.CustomResource.Spec.Database.Kind) + ".git"
	} else {
		appSettings.RepositoryUrl += repoUrl + ".git"
	}
}

func (businessApplication BusinessApplication) Update() {

}

func (businessApplication BusinessApplication) Delete() {

}
