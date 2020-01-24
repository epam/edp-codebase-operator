package util

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

func getUserSettingsConfigMap(client *coreV1Client.CoreV1Client, namespace string) (*model.UserSettings, error) {
	us, err := client.ConfigMaps(namespace).Get("edp-config", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	vcsIntegrationEnabled, err := strconv.ParseBool(us.Data["vcs_integration_enabled"])
	if err != nil {
		return nil, err
	}
	perfIntegrationEnabled, err := strconv.ParseBool(us.Data["perf_integration_enabled"])
	if err != nil {
		return nil, err
	}
	return &model.UserSettings{
		DnsWildcard:            us.Data["dns_wildcard"],
		EdpName:                us.Data["edp_name"],
		EdpVersion:             us.Data["edp_version"],
		PerfIntegrationEnabled: perfIntegrationEnabled,
		VcsGroupNameUrl:        us.Data["vcs_group_name_url"],
		VcsIntegrationEnabled:  vcsIntegrationEnabled,
		VcsSshPort:             us.Data["vcs_ssh_port"],
		VcsToolName:            model.VCSTool(us.Data["vcs_tool_name"]),
	}, nil
}

func getGerritSettingsConfigMap(client *coreV1Client.CoreV1Client, namespace string) (*model.GerritSettings, error) {
	gs, err := client.ConfigMaps(namespace).Get("gerrit", metav1.GetOptions{})
	sshPort, err := strconv.ParseInt(gs.Data["sshPort"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &model.GerritSettings{
		Config:            gs.Data["config"],
		ReplicationConfig: gs.Data["replication.config"],
		SshPort:           int32(sshPort),
	}, nil
}

func GetConfigSettings(client *coreV1Client.CoreV1Client, namespace string) (*model.GerritSettings, *model.UserSettings, error) {
	log.Info("Start getting Gerrit Settings Config Map...")
	gs, err := getGerritSettingsConfigMap(client, namespace)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Start getting User Settings Config Map...")
	us, err := getUserSettingsConfigMap(client, namespace)
	if err != nil {
		return nil, nil, err
	}
	return gs, us, nil
}

func GetVcsBasicAuthConfig(c coreV1Client.CoreV1Client, namespace string, secretName string) (string, string, error) {
	log.Info("Start getting secret", "name", secretName)
	vcsCredentialsSecret, err := c.
		Secrets(namespace).
		Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return "", "", err
	}
	return string(vcsCredentialsSecret.Data["username"]), string(vcsCredentialsSecret.Data["password"]), nil
}

func GetGitServer(c client.Client, codebaseName, gitServerName, namespace string) (*model.GitServer, error) {
	gitSec, err := getGitServerCR(c, gitServerName, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting Git Server CR for %v codebase", codebaseName)
	}

	gs, err := model.ConvertToGitServer(*gitSec)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while converting request Git Server to DTO for %v codebase",
			codebaseName)
	}
	return gs, nil
}

func getGitServerCR(c client.Client, name, namespace string) (*edpv1alpha1.GitServer, error) {
	log.Info("Start fetching GitServer resource from k8s", "name", name, "namespace", namespace)
	instance := &edpv1alpha1.GitServer{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, instance); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "Git Server %v doesn't exist in k8s.", name)
		}
		return nil, err
	}
	log.Info("Git Server instance has been received", "name", name)
	return instance, nil
}

func GetSecret(c coreV1Client.CoreV1Client, secretName, namespace string) (*v1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", secretName, "namespace", namespace)
	secret, err := c.
		Secrets(namespace).
		Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return nil, err
	}
	log.Info("Secret has been fetched", "secret name", secretName, "namespace", namespace)
	return secret, nil
}
