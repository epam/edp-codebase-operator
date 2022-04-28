package util

import (
	"context"
	"fmt"
	"os"
	"strconv"

	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	watchNamespaceEnvVar = "WATCH_NAMESPACE"
	debugModeEnvVar      = "DEBUG_MODE"
)

func GetUserSettings(client client.Client, namespace string) (*model.UserSettings, error) {
	us := &coreV1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      "edp-config",
	}, us)
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
	if gs.Spec.SshPort == 0 {
		return nil, errors.New("ssh port is zero or not defined in gerrit GitServer CR")
	}
	return getInt32P(gs.Spec.SshPort), nil
}

func getInt32P(val int32) *int32 {
	return &val
}

func GetVcsBasicAuthConfig(c client.Client, namespace string, secretName string) (string, string, error) {
	log.Info("Start getting secret", "name", secretName)
	secret := &coreV1.Secret{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)
	if err != nil {
		return "", "", errors.Wrapf(err, "Unable to get secret %v", secretName)
	}
	if string(secret.Data["username"]) == "" || string(secret.Data["password"]) == "" {
		return "", "", errors.Errorf("username/password keys are not defined in Secret %v ", secretName)
	}
	return string(secret.Data["username"]), string(secret.Data["password"]), nil
}

func GetGitServer(c client.Client, name, namespace string) (*model.GitServer, error) {
	gitReq, err := getGitServerCR(c, name, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting %v Git Server CR", name)
	}

	gs := model.ConvertToGitServer(*gitReq)
	return gs, nil
}

func getGitServerCR(c client.Client, name, namespace string) (*edpv1alpha1.GitServer, error) {
	log.Info("Start fetching GitServer resource from k8s", "name", name, "namespace", namespace)
	instance := &edpv1alpha1.GitServer{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, instance); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "GitServer %v doesn't exist in k8s.", name)
		}
		return nil, errors.Wrapf(err, "Unable to get GitServer %v", name)
	}
	log.Info("Git Server instance has been received", "name", name)
	return instance, nil
}

func GetSecret(c client.Client, secretName, namespace string) (*coreV1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", secretName, "namespace", namespace)
	secret := &coreV1.Secret{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get secret %v", secretName)
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
		return nil, errors.Wrapf(err, "Unable to get Codebase %v", name)
	}

	return instance, nil
}

func GetEdpComponent(c client.Client, name, namespace string) (*v1alpha1.EDPComponent, error) {
	ec := &v1alpha1.EDPComponent{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, ec)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get EDPComponent %v", name)
	}
	return ec, nil
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// GetDebugMode returns the debug mode value
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, err
	}
	return b, nil
}
