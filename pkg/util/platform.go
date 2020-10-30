package util

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

func GetUserSettings(client *coreV1Client.CoreV1Client, namespace string) (*model.UserSettings, error) {
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

func GetGerritPort(c client.Client, namespace string) (*int32, error) {
	gs, err := getGitServerCR(c, "gerrit", namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting %v Git Server CR", "gerrit")
	}
	return getInt32P(gs.Spec.SshPort), nil
}

func getInt32P(val int32) *int32 {
	return &val
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

func GetGitServer(c client.Client, name, namespace string) (*model.GitServer, error) {
	gitReq, err := getGitServerCR(c, name, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting %v Git Server CR", name)
	}

	gs, err := model.ConvertToGitServer(*gitReq)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while converting request %v Git Server to DTO",
			name)
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

func GetCodebase(client client.Client, name, namespace string) (*edpv1alpha1.Codebase, error) {
	instance := &edpv1alpha1.Codebase{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func GetSecretData(client client.Client, name, namespace string) (*coreV1.Secret, error) {
	s := &coreV1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetEdpComponent(c client.Client, name, namespace string) (*v1alpha1.EDPComponent, error) {
	ec := &v1alpha1.EDPComponent{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, ec)
	if err != nil {
		return nil, err
	}
	return ec, nil
}
