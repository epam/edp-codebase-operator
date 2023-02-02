package gerrit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

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

type ReplicationConfigParams struct {
	Name      string
	VcsSshUrl string
}

const (
	ReplicationConfigTemplateName = "replication-conf.tmpl"
)

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

func SshInit(port int32, idrsa, host string, logger logr.Logger) (*SSHClient, error) {
	pubkey, err := ssh.ParsePrivateKey([]byte(idrsa))
	if err != nil {
		return nil, fmt.Errorf("failed to get Public Key from Private one: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: "project-creator",
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

	logger.Info("SSH Client has been initialized", "host", host, "port", port)

	return &cl, nil
}

func CheckProjectExist(port int32, idrsa, host, appName string, logger logr.Logger) (*bool, error) {
	var raw map[string]interface{}

	command := "gerrit ls-projects --format json"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, idrsa, host, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init ssh: %w", err)
	}

	outputCmd, err := cl.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run ssh command: %w", err)
	}

	err = json.Unmarshal(outputCmd, &raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	raw["count"] = 1
	_, isExist := raw[appName]

	return &isExist, nil
}

func CreateProject(port int32, idrsa, host, appName string, logger logr.Logger) error {
	command := fmt.Sprintf("gerrit create-project %v", appName)
	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, idrsa, host, logger)
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
func SetHeadToBranch(port int32, idrsa, host, appName, branchName string, logger logr.Logger) error {
	command := fmt.Sprintf("gerrit set-head %v --new-head %v", appName, branchName)
	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, idrsa, host, logger)
	if err != nil {
		return err
	}

	_, err = cl.RunCommand(cmd)

	return err
}

func AddRemoteLinkToGerrit(repoPath, host string, port int32, appName string, logger logr.Logger) error {
	remoteUrl := fmt.Sprintf("ssh://%v:%v/%v", host, port, appName)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open Git directory: %w", err)
	}

	err = r.DeleteRemote("origin")
	if err != nil && !errors.Is(err, git.ErrRemoteNotFound) {
		return fmt.Errorf("failed to delete remote origin: %w", err)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteUrl},
	})
	if err != nil {
		return fmt.Errorf("failed to create remote origin: %w", err)
	}

	logger.Info("Remote link has been added", "repoPath", repoPath, "host", host, "port", port,
		"appName", appName)

	return nil
}

func generateReplicationConfig(templatePath, templateName string, params ReplicationConfigParams) (string, error) {
	log.Printf("Start generation replication config by template path: %v, template name: %v, with params: %+v",
		templatePath, templateName, params)

	replicationFullPath := fmt.Sprintf("%v/%v", templatePath, templateName)

	var renderedTemplate bytes.Buffer

	tmpl, err := template.New(templateName).
		ParseFiles(replicationFullPath)
	if err != nil {
		log.Printf("Error has been occured during the parcing template by full path: %v", replicationFullPath)
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	err = tmpl.Execute(&renderedTemplate, params)
	if err != nil {
		log.Printf("failed to render replication config: %v", err)

		return "", fmt.Errorf("failed to process template: %w", err)
	}

	log.Printf("Replication config has been generated successsfully"+
		" by template path: %v, template name: %v, with params: %+v", templatePath, templateName, params)

	return renderedTemplate.String(), nil
}

func SetupProjectReplication(c client.Client, sshPort int32, host, idrsa, codebaseName, namespace,
	vcsSshUrl string, logger logr.Logger,
) error {
	logger.Info("Start setup project replication for app", "codebase", codebaseName)

	ctx := context.Background()

	replicaConfigNew, err := generateReplicationConfig(
		fmt.Sprintf("%v/templates/gerrit", util.GetAssetsDir()),
		ReplicationConfigTemplateName, ReplicationConfigParams{
			Name:      codebaseName,
			VcsSshUrl: vcsSshUrl,
		})
	if err != nil {
		return fmt.Errorf("failed to generate replication config: %w", err)
	}

	gerritSettings := &v1.ConfigMap{}

	err = c.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      "gerrit",
	}, gerritSettings)
	if err != nil {
		return fmt.Errorf("failed to get %v config map: %w", "gerrit", err)
	}

	replicaConfig := gerritSettings.Data["replication.config"]
	if replicaConfig == "" {
		return errors.New("replication.config key is missing in gerrit ConfigMap")
	}

	gerritSettings.Data["replication.config"] = fmt.Sprintf("%v\n%v", replicaConfig, replicaConfigNew)

	err = c.Update(ctx, gerritSettings)
	if err != nil {
		log.Printf("failed to update config map with replication config: %v", err)

		return fmt.Errorf("failed to update 'ConfigMap' resource %q: %w", gerritSettings.Name, err)
	}

	// TODO: refactor.
	const waitDuration = 5 * time.Second

	log.Println("Waiting for gerrit replication config map appears in gerrit pod. Sleeping for 5 seconds...")
	time.Sleep(waitDuration)

	err = reloadReplicationPlugin(sshPort, idrsa, host, logger)
	if err != nil {
		log.Printf("failed to reload replication plugin: %v", err)

		return err
	}

	log.Printf("Replication configuration has been finished for app %v", codebaseName)

	return nil
}

func reloadReplicationPlugin(port int32, idrsa, host string, logger logr.Logger) error {
	command := "gerrit plugin reload replication"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cl, err := SshInit(port, idrsa, host, logger)
	if err != nil {
		return err
	}

	_, err = cl.RunCommand(cmd)
	if err != nil {
		return err
	}

	logger.Info("Gerrit replication plugin has been reloaded", "Host", host, "Port", port)

	return nil
}
