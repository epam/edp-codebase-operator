package impl

import (
	"business-app-handler-controller/models"
	edpv1alpha1 "business-app-handler-controller/pkg/apis/edp/v1alpha1"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"business-app-handler-controller/pkg/perfTool"
	"business-app-handler-controller/pkg/settings"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	if appSettings.UserSettings.VcsIntegrationEnabled == "true" {
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

	if appSettings.UserSettings.PerfIntegrationEnabled == "true" {
		perfSecret := perfTool.GetPerfCredentials(*clientSet, businessApplication.CustomResource.Namespace)
		perfSettings := perfTool.GetPerfSettings(*clientSet, businessApplication.CustomResource.Namespace)
		perfToken := perfTool.GetPerfToken(perfSettings["perf_web_url"], string(perfSecret["username"]), string(perfSecret["password"]))
		fmt.Println(perfToken) //ToDo: remove when perfToken in use
	} else {
		log.Printf("Perf integration isn't enabled")
	}

	appSettings.GerritPrivateKey, appSettings.GerritPublicKey = settings.GetGerritCredentials(*clientSet, businessApplication.CustomResource.Namespace)

	log.Printf("Retrieving settings has been finished.")

	settings.CreateGerritPrivateKey(appSettings.GerritPrivateKey, appSettings.GerritKeyPath)
	settings.CreateSshConfig(appSettings)

	switch strings.ToLower(businessApplication.CustomResource.Spec.Strategy) {
	case strings.ToLower(allowedAppSettings["add_repo_strategy"][0]):
		fmt.Println("Strategy create")
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
}

func (businessApplication BusinessApplication) Update() {

}

func (businessApplication BusinessApplication) Delete() {

}
