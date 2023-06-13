package gitserver

import (
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type gitSshData struct {
	Host string
	User string
	Key  string
	Port int32
}

func checkConnectionToGitServer(c client.Client, gitServer *model.GitServer, log logr.Logger) (bool, error) {
	log.Info("Start CheckConnectionToGitServer method", "Git host", gitServer.GitHost)

	sshSecret, err := util.GetSecret(c, gitServer.NameSshKeySecret, gitServer.Namespace)
	if err != nil {
		return false, fmt.Errorf("failed to get %v secret: %w", gitServer.NameSshKeySecret, err)
	}

	sshData := extractSshData(gitServer, sshSecret)

	log.Info("Data from request is extracted", "host", sshData.Host, "port", sshData.Port)

	accessible, err := isGitServerAccessible(sshData, log)
	if err != nil {
		return false, fmt.Errorf("an error has occurred while checking connection to git server: %w", err)
	}

	log.Info("Git server", "accessible", accessible)

	return accessible, nil
}

func isGitServerAccessible(data gitSshData, log logr.Logger) (bool, error) {
	log.Info("Start executing IsGitServerAccessible method to check connection to server", "host", data.Host)

	sshClient, err := sshInitFromSecret(data, log)
	if err != nil {
		log.Info(fmt.Sprintf("An error has occurred while initing SSH client. Check data in Git Server resource and secret: %v", err))
		return false, err
	}

	var (
		s *ssh.Session
		c *ssh.Client
	)

	if s, c, err = sshClient.NewSession(); err != nil {
		log.Info(fmt.Sprintf("An error has occurred while connecting to server. Check data in Git Server resource and secret: %v", err))
		return false, nil
	}

	defer util.CloseWithLogOnErr(log, s, "failed to close ssh client session")
	defer util.CloseWithLogOnErr(log, c, "failed to close ssh client connection")

	return s != nil && c != nil, nil
}

func extractSshData(gitServer *model.GitServer, secret *corev1.Secret) gitSshData {
	return gitSshData{
		Host: gitServer.GitHost,
		User: gitServer.GitUser,
		Key:  string(secret.Data[util.PrivateSShKeyName]),
		Port: gitServer.SshPort,
	}
}

func sshInitFromSecret(data gitSshData, logger logr.Logger) (*gerrit.SSHClient, error) {
	sshAuth, err := publicKey(data.Key)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: data.User,
		Auth: []ssh.AuthMethod{
			sshAuth,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	cl := &gerrit.SSHClient{
		Config: sshConfig,
		Host:   data.Host,
		Port:   data.Port,
	}

	logger.Info("SSH Client has been initialized", "host", data.Host, "port", data.Port)

	return cl, nil
}

func publicKey(key string) (ssh.AuthMethod, error) {
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}
