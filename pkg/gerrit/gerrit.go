package gerrit

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
)

// Client is an interface for Gerrit client.
type Client interface {
	CreateProject(port int32, sshPrivateKey, host, user, appName string, logger logr.Logger) error
	CheckProjectExist(port int32, sshPrivateKey, host, user, appName string, logger logr.Logger) (bool, error)
	SetHeadToBranch(port int32, sshPrivateKey, host, user, appName, branchName string, logger logr.Logger) error
}

type SSHGerritClient struct {
}

type SSHCommand struct {
	Path   string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type SSHClient struct {
	Config *ssh.ClientConfig
	Host   string
	Port   int32
}

func (s *SSHClient) RunCommand(cmd *SSHCommand) ([]byte, error) {
	var (
		session    *ssh.Session
		connection *ssh.Client
		err        error
	)

	if session, connection, err = s.NewSession(); err != nil {
		return nil, err
	}

	defer func() {
		if deferErr := session.Close(); deferErr != nil {
			err = deferErr
		}

		if deferErr := connection.Close(); deferErr != nil {
			err = deferErr
		}
	}()

	commandOutput, err := session.Output(cmd.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get ssh STD out: %w", err)
	}

	return commandOutput, nil
}

func (s *SSHClient) NewSession() (*ssh.Session, *ssh.Client, error) {
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), s.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %s", err)
	}

	return session, connection, nil
}

func SshInit(port int32, sshPrivateKey, host, user string, logger logr.Logger) (*SSHClient, error) {
	pubkey, err := ssh.ParsePrivateKey([]byte(sshPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get Public Key from Private one: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(pubkey),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		}),
	}
	cl := SSHClient{
		Config: sshConfig,
		Host:   host,
		Port:   port,
	}

	logger.Info("SSH Client has been initialized", "host", host, "port", port, "user", user)

	return &cl, nil
}

func (*SSHGerritClient) CheckProjectExist(port int32, sshPrivateKey, host, user, appName string, logger logr.Logger) (bool, error) {
	var raw map[string]interface{}

	command := "gerrit ls-projects --format json"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, sshPrivateKey, host, user, logger)
	if err != nil {
		return false, fmt.Errorf("failed to init ssh: %w", err)
	}

	outputCmd, err := cl.RunCommand(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to run ssh command: %w", err)
	}

	err = json.Unmarshal(outputCmd, &raw)
	if err != nil {
		return false, fmt.Errorf("failed to decode json: %w", err)
	}

	raw["count"] = 1
	_, isExist := raw[appName]

	return isExist, nil
}

func (*SSHGerritClient) CreateProject(port int32, sshPrivateKey, host, user, appName string, logger logr.Logger) error {
	command := fmt.Sprintf("gerrit create-project %v", appName)
	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, sshPrivateKey, host, user, logger)
	if err != nil {
		return err
	}

	_, err = cl.RunCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

// SetHeadToBranch sets remote git HEAD to specific branch using ssh Gerrit command.
func (*SSHGerritClient) SetHeadToBranch(port int32, sshPrivateKey, host, user, appName, branchName string, logger logr.Logger) error {
	command := fmt.Sprintf("gerrit set-head %v --new-head %v", appName, branchName)
	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, sshPrivateKey, host, user, logger)
	if err != nil {
		return err
	}

	_, err = cl.RunCommand(cmd)

	return err
}
