package gerrit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"html/template"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
	"net"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"time"
)

var logger = logf.Log.WithName("git-server-service")

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

func (client *SSHClient) RunCommand(cmd *SSHCommand) ([]byte, error) {
	var session *ssh.Session
	var connection *ssh.Client
	var err error

	if session, connection, err = client.NewSession(); err != nil {
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
		return nil, err
	}

	return commandOutput, err
}

func (client *SSHClient) NewSession() (*ssh.Session, *ssh.Client, error) {
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", client.Host, client.Port), client.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %s", err)
	}

	return session, connection, nil
}

func PublicKeyFile(idrsa string) ssh.AuthMethod {
	key, err := ssh.ParsePrivateKey([]byte(idrsa))
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func SshInit(port int32, idrsa, host string) (SSHClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: "project-creator",
		Auth: []ssh.AuthMethod{
			PublicKeyFile(idrsa),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}

	client := &SSHClient{
		Config: sshConfig,
		Host:   host,
		Port:   port,
	}
	log.Printf("SSH Client has been initialized: Host: %v Port: %v", host, port)

	return *client, nil
}

func CheckProjectExist(port int32, idrsa, host, appName string) (*bool, error) {
	var raw map[string]interface{}

	command := "gerrit ls-projects --format json"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	client, err := SshInit(port, idrsa, host)
	if err != nil {
		return nil, err
	}

	outputCmd, err := client.RunCommand(cmd)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(outputCmd, &raw)
	if err != nil {
		return nil, err
	}

	raw["count"] = 1
	_, isExist := raw[appName]

	return &isExist, nil
}

func CreateProject(port int32, idrsa, host, appName string) error {
	command := fmt.Sprintf("gerrit create-project %v", appName)
	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	client, err := SshInit(port, idrsa, host)
	if err != nil {
		return err
	}

	_, err = client.RunCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

func AddRemoteLinkToGerrit(repoPath string, host string, port int32, appName string) error {
	remoteUrl := fmt.Sprintf("ssh://project-creator@%v:%v/%v", host, port, appName)
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		log.Println(err)
		return err
	}
	err = r.DeleteRemote("origin")
	if err != nil {
		log.Println(err)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteUrl},
	})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Remote link has been added: repoPath: %v Host: %v Port: %v AppName: %v",
		repoPath, host, port, appName)
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
		return "", err
	}
	err = tmpl.Execute(&renderedTemplate, params)
	if err != nil {
		log.Printf("Unable to render replication config: %v", err)
		return "", err
	}
	log.Printf("Replication config has been generated successsfully"+
		" by template path: %v, template name: %v, with params: %+v", templatePath, templateName, params)
	return renderedTemplate.String(), nil
}

func SetupProjectReplication(client coreV1Client.CoreV1Client, sshPort int32, host, idrsa, codebaseName, namespace, vcsSshUrl string) error {

	log.Printf("Start setup project replication for app: %v", codebaseName)
	replicaConfigNew, err := generateReplicationConfig(
		util.GerritTemplates, ReplicationConfigTemplateName, ReplicationConfigParams{
			Name:      codebaseName,
			VcsSshUrl: vcsSshUrl,
		})

	gerritSettings, err := client.ConfigMaps(namespace).Get("gerrit", metav1.GetOptions{})
	replicaConfig := gerritSettings.Data["replication.config"]
	gerritSettings.Data["replication.config"] = fmt.Sprintf("%v\n%v", replicaConfig, replicaConfigNew)
	result, err := client.ConfigMaps(namespace).Update(gerritSettings)
	if err != nil {
		log.Printf("Unable to update config map with replication config: %v", err)
		return err
	}
	log.Println(result)

	log.Println("Waiting for gerrit replication config map appears in gerrit pod. Sleeping for 90 seconds...")
	time.Sleep(90 * time.Second)

	err = reloadReplicationPlugin(sshPort, idrsa, host)
	if err != nil {
		log.Printf("Unable to reload replication plugin: %v", err)
		return err
	}
	log.Printf("Replication configuration has been finished for app %v", codebaseName)

	return nil
}

func reloadReplicationPlugin(port int32, idrsa, host string) error {
	command := "gerrit plugin reload replication"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	client, err := SshInit(port, idrsa, host)
	if err != nil {
		return err
	}

	_, err = client.RunCommand(cmd)
	if err != nil {
		return err
	}

	log.Printf("Gerrit replication plugin has been reloaded Host: %v Port: %v", host, port)
	return nil
}
