package service

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("git-server-service")

type GitServerService struct {
	ClientSet *openshift.ClientSet
}

type GerritJenkinsSshSecret struct {
	SshPub string
}

const (
	EdpPostfix = "edp-cicd"
	KeyName    = "id_rsa"
)

func (s *GitServerService) CheckConnectionToGitServer(gitServer model.GitServer) (bool, error) {
	log.Info("Start CheckConnectionToGitServer method", "Git host", gitServer.GitHost)

	sshSecret, err := s.getSecret(gitServer.NameSshKeySecret, gitServer.Tenant)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("an error has occurred  while getting %v secret", gitServer.NameSshKeySecret))
	}

	gitSshData := extractSshData(gitServer, sshSecret)

	log.Info("Extracted data from request", "data", gitSshData)

	accessible := gerrit.IsGitServerAccessible(gitSshData)

	log.Info("Git server", "accessible", accessible)
	return accessible, nil
}

func (s *GitServerService) getSecret(secretName, namespace string) (*v1.Secret, error) {
	secret, err := s.ClientSet.CoreClient.
		Secrets(fmt.Sprintf("%s-%s", namespace, EdpPostfix)).
		Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return nil, err
	}

	return secret, nil
}

func extractSshData(gitServer model.GitServer, secret *v1.Secret) gerrit.GitSshData {
	return gerrit.GitSshData{
		Host: gitServer.GitHost,
		User: gitServer.GitUser,
		Key:  string(secret.Data[KeyName]),
		Port: gitServer.SshPort,
	}
}
