package settings

import (
	"business-application-operator/models"
	ClientSet "business-application-operator/pkg/openshift"
	"html/template"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

func RetryFunc(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(sleep)
		log.Printf("Retrying after error: %s", err)
	}
	return err
}

func GetUserSettingsConfigMap(clientSet ClientSet.OpenshiftClientSet, namespace string) models.UserSettings {
	userSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("user-settings", metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return models.UserSettings{
		DnsWildcard:            userSettings.Data["dns_wildcard"],
		EdpName:                userSettings.Data["edp_name"],
		EdpVersion:             userSettings.Data["edp_version"],
		PerfIntegrationEnabled: userSettings.Data["perf_integration_enabled"],
		VcsGroupNameUrl:        userSettings.Data["vcs_group_name_url"],
		VcsIntegrationEnabled:  userSettings.Data["vcs_integration_enabled"],
		VcsSshPort:             userSettings.Data["vcs_ssh_port"],
		VcsToolName:            userSettings.Data["vcs_tool_name"],
	}
}

func GetGerritSettingsConfigMap(clientSet ClientSet.OpenshiftClientSet, namespace string) models.GerritSettings {
	gerritSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("gerrit", metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return models.GerritSettings{
		Config:            gerritSettings.Data["config"],
		ReplicationConfig: gerritSettings.Data["replication.config"],
		SshPort:           gerritSettings.Data["sshPort"],
	}
}

func GetJenkinsCreds(clientSet ClientSet.OpenshiftClientSet, namespace string) (string, string) {
	jenkinsTokenSecret, err := clientSet.CoreClient.Secrets(namespace).Get("jenkins-token", metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return string(jenkinsTokenSecret.Data["token"]), string(jenkinsTokenSecret.Data["username"])
}

func GetVcsCredentials(clientSet ClientSet.OpenshiftClientSet, namespace string) (string, string) {
	vcsAutouserSecret, err := clientSet.CoreClient.Secrets(namespace).Get("vcs-autouser", metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return string(vcsAutouserSecret.Data["ssh-privatekey"]), string(vcsAutouserSecret.Data["username"])
}

func GetGerritCredentials(clientSet ClientSet.OpenshiftClientSet, namespace string) (string, string) {
	vcsAutouserSecret, err := clientSet.CoreClient.Secrets(namespace).Get("gerrit-project-creator", metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return string(vcsAutouserSecret.Data["id_rsa"]), string(vcsAutouserSecret.Data["id_rsa.pub"])
}

func CreateGerritPrivateKey(privateKey string, path string) () {
	err := ioutil.WriteFile(path, []byte(privateKey), 0400)
	if err != nil {
		log.Fatalf("Unable to write the key: %v", err)
	}
}

func CreateSshConfig(appSettings models.AppSettings) {
	if _, err := os.Stat("~/.ssh"); os.IsNotExist(err) {
		os.Mkdir("~/.ssh", 0644)
	}

	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	tmpl := template.Must(template.New("config.tmpl").ParseFiles(workDir + "/templates/ssh/config.tmpl"))
	f, err := os.Create("~/.ssh/config")
	if err != nil {
		log.Fatalf("Cannot write SSH config to the file: %v", err)
	}
	err = tmpl.Execute(f, appSettings)
	if err != nil {
		log.Print("execute: ", err)
		return
	}
	defer f.Close()
}

func IsFrameworkMultiModule(name string) bool {
	regexpMultiModuleFramework := regexp.MustCompile(`\(([^)]+)\)`)
	match := regexpMultiModuleFramework.FindAllStringSubmatch(name, -1)
	if match == nil {
		return false
	} else {
		return true
	}
}

func AddFrameworkMultiModulePostfix(name string) string {
	regexpMultiModuleFramework := regexp.MustCompile(`\(([^)]+)\)`)
	return regexpMultiModuleFramework.ReplaceAllString(strings.ToLower(name), "-multimodule")
}
