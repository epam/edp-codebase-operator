package gitserver

import (
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"

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

// checkGitServerConnection checks connection to Git server. If connection is not established, returns error.
func checkGitServerConnection(data gitSshData, log logr.Logger) error {
	log.Info("Start executing IsGitServerAccessible method to check connection to server", "host", data.Host)

	sshClient, err := sshInitFromSecret(data, log)
	if err != nil {
		return fmt.Errorf("failed to initialize ssh client: %w", err)
	}

	var (
		s *ssh.Session
		c *ssh.Client
	)

	if s, c, err = sshClient.NewSession(); err != nil {
		return fmt.Errorf("failed to create ssh session: %w", err)
	}

	defer util.CloseWithLogOnErr(log, c, "failed to close ssh client connection")

	if s != nil {
		defer util.CloseWithLogOnErr(log, s, "failed to close ssh client session")
	}

	return nil
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
