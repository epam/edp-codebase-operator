package settings

import (
	"business-app-handler-controller/models"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"strconv"
)

func CreateWorkdir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0744)
		}
	}
	return nil
}

func GetUserSettingsConfigMap(clientSet ClientSet.ClientSet, namespace string) (*models.UserSettings, error) {
	userSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("user-settings", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get user settings configmap: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	vcsIntegrationEnabled, err := strconv.ParseBool(userSettings.Data["vcs_integration_enabled"])
	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	perfIntegrationEnabled, err := strconv.ParseBool(userSettings.Data["perf_integration_enabled"])
	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	return &models.UserSettings{
		DnsWildcard:            userSettings.Data["dns_wildcard"],
		EdpName:                userSettings.Data["edp_name"],
		EdpVersion:             userSettings.Data["edp_version"],
		PerfIntegrationEnabled: perfIntegrationEnabled,
		VcsGroupNameUrl:        userSettings.Data["vcs_group_name_url"],
		VcsIntegrationEnabled:  vcsIntegrationEnabled,
		VcsSshPort:             userSettings.Data["vcs_ssh_port"],
		VcsToolName:            models.VCSTool(userSettings.Data["vcs_tool_name"]),
	}, nil
}

func getEnvSettingsConfigMap(clientSet ClientSet.ClientSet, namespace string) (*string, error) {
	projectSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("project-settings", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get env settings configmap: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	envSettings := projectSettings.Data["env.settings.json"]

	return &envSettings, nil
}

func GetEnvSettings(clientSet ClientSet.ClientSet, namespace string) ([]models.EnvSettings, error) {
	var envSettings []models.EnvSettings

	settings, err := getEnvSettingsConfigMap(clientSet, namespace)

	err = json.Unmarshal([]byte(*settings), &envSettings)
	if err != nil {
		return nil, err
	}

	if len(envSettings) == 0 {
		return nil, err
	}

	return envSettings, nil
}

func GetGerritSettingsConfigMap(clientSet ClientSet.ClientSet, namespace string) (*models.GerritSettings, error) {
	gerritSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("gerrit", metav1.GetOptions{})
	sshPort, err := strconv.ParseInt(gerritSettings.Data["sshPort"], 10, 64)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get Gerrit settings configmap: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	return &models.GerritSettings{
		Config:            gerritSettings.Data["config"],
		ReplicationConfig: gerritSettings.Data["replication.config"],
		SshPort:           sshPort,
	}, nil
}

func GetJenkinsCreds(clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	jenkinsTokenSecret, err := clientSet.CoreClient.Secrets(namespace).Get("jenkins-token", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return "", "", errors.New(errorMsg)
	}
	return string(jenkinsTokenSecret.Data["token"]), string(jenkinsTokenSecret.Data["username"]), nil
}

func GetVcsCredentials(clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	vcsAutouserSecret, err := clientSet.CoreClient.Secrets(namespace).Get("vcs-autouser", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get VCS credentials: %v", err)
		log.Println(errorMsg)
		return "", "", errors.New(errorMsg)
	}
	return string(vcsAutouserSecret.Data["ssh-privatekey"]), string(vcsAutouserSecret.Data["username"]), nil
}

func GetGerritCredentials(clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	vcsAutouserSecret, err := clientSet.CoreClient.Secrets(namespace).Get("gerrit-project-creator", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get gerrit credentials: %v", err)
		log.Println(errorMsg)
		return "", "", errors.New(errorMsg)
	}
	return string(vcsAutouserSecret.Data["id_rsa"]), string(vcsAutouserSecret.Data["id_rsa.pub"]), nil
}

func CreateGerritPrivateKey(privateKey string, path string) error {
	err := ioutil.WriteFile(path, []byte(privateKey), 400)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to write the Gerrit ssh key: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	return nil
}

func CreateSshConfig(appSettings models.AppSettings) error {
	sshPath := "/home/business-app-handler-controller/.ssh"
	if _, err := os.Stat(sshPath); os.IsNotExist(err) {
		err = os.MkdirAll(sshPath, 0744)
		if err != nil {
			return err
		}
	}

	tmpl, err := template.New("config.tmpl").ParseFiles("/usr/local/bin/templates/ssh/config.tmpl")
	if err != nil {
		return err
	}
	f, err := os.Create(sshPath + "/config")
	if err != nil {
		log.Printf("Cannot write SSH config to the file: %v", err)
		return err
	}
	err = tmpl.Execute(f, appSettings)
	if err != nil {
		log.Printf("execute: %v", err)
		return err
	}
	defer f.Close()
	return nil
}

func GetVcsBasicAuthConfig(clientSet ClientSet.ClientSet, namespace string, secretName string) (string, string, error) {
	vcsCredentialsSecret, err := clientSet.CoreClient.Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return "", "", nil
	}
	err = DeleteTempVcsSecret(clientSet, namespace, secretName)
	if err != nil {
		return "", "", err
	}
	return string(vcsCredentialsSecret.Data["username"]), string(vcsCredentialsSecret.Data["password"]), nil
}

func DeleteTempVcsSecret(clientSet ClientSet.ClientSet, namespace string, secretName string) error {
	err := clientSet.CoreClient.Secrets(namespace).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		errorMsg := fmt.Sprintf("Unable to delete temp secret: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	return nil
}
