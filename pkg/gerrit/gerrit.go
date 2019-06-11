package gerrit

import (
	"bytes"
	"codebase-operator/models"
	ClientSet "codebase-operator/pkg/openshift"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	sshgit "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"html/template"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net"
	"os"
	"strconv"
	"time"
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
	Port   int64
}

type ReplicationConfigParams struct {
	Name      string
	VcsSshUrl string
}

const (
	ReplicationConfigTemplateName = "replication-conf.tmpl"
	TemplatesPath                 = "/usr/local/bin/templates/gerrit"
)

func (client *SSHClient) RunCommand(cmd *SSHCommand) ([]byte, error) {
	var session *ssh.Session
	var connection *ssh.Client
	var err error

	if session, connection, err = client.newSession(); err != nil {
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

func (client *SSHClient) newSession() (*ssh.Session, *ssh.Client, error) {
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

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func SshInit(keyPath string, host string, port int64) (SSHClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: "project-creator",
		Auth: []ssh.AuthMethod{
			PublicKeyFile(keyPath),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}

	client := &SSHClient{
		Config: sshConfig,
		Host:   host,
		Port:   port,
	}
	log.Printf("SSH Client has been initialized: keyPath: %v Host: %v Port: %v", keyPath, host, port)

	return *client, nil
}

func CheckProjectExist(keyPath string, host string, port int64, appName string) (*bool, error) {
	var raw map[string]interface{}

	command := "gerrit ls-projects --format json"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	client, err := SshInit(keyPath, host, port)
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

func CreateProject(keyPath string, host string, port int64, appName string) error {
	command := fmt.Sprintf("gerrit create-project %v", appName)
	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	client, err := SshInit(keyPath, host, port)
	if err != nil {
		return err
	}

	_, err = client.RunCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

func AddRemoteLinkToGerrit(repoPath string, host string, port int64, appName string) error {
	remoteUrl := fmt.Sprintf("ssh://project-creator@%v:%v/%v", host, strconv.FormatInt(port, 10), appName)
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

func Auth(keyPath string) (transport.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	log.Println("Private key has been read")

	signer, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	sshgitPublicKeys := new(sshgit.PublicKeys)
	sshgitPublicKeys.User = "project-creator"
	sshgitPublicKeys.Signer = signer
	sshgitPublicKeys.HostKeyCallbackHelper = sshgit.HostKeyCallbackHelper{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return sshgitPublicKeys, nil
}

func PushToGerrit(repoPath string, keyPath string) error {
	r, err := git.PlainOpen(repoPath)
	log.Printf("Repo with project has been opened: %v", repoPath)
	if err != nil {
		return err
	}
	auth, err := Auth(keyPath)
	if err != nil {
		return err
	}

	gitOptions := new(git.PushOptions)
	gitOptions.RemoteName = "origin"
	gitOptions.RefSpecs = []config.RefSpec{
		"refs/heads/*:refs/heads/*",
		"refs/tags/*:refs/tags/*",
	}
	gitOptions.Auth = auth

	err = r.Push(gitOptions)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Pushed to gerrit repo %v", repoPath)
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

func SetupProjectReplication(codebaseSettings models.CodebaseSettings, clientSet ClientSet.ClientSet) error {
	log.Printf("Start setup project replication for app: %v", codebaseSettings.Name)
	replicaConfigNew, err := generateReplicationConfig(
		TemplatesPath, ReplicationConfigTemplateName, ReplicationConfigParams{
			Name:      codebaseSettings.Name,
			VcsSshUrl: codebaseSettings.VcsSshUrl,
		})

	gerritSettings, err := clientSet.CoreClient.ConfigMaps(codebaseSettings.CicdNamespace).Get("gerrit", metav1.GetOptions{})
	replicaConfig := gerritSettings.Data["replication.config"]
	gerritSettings.Data["replication.config"] = fmt.Sprintf("%v\n%v", replicaConfig, replicaConfigNew)
	result, err := clientSet.CoreClient.ConfigMaps(codebaseSettings.CicdNamespace).Update(gerritSettings)
	if err != nil {
		log.Printf("Unable to update config map with replication config: %v", err)
		return err
	}
	log.Println(result)

	log.Println("Waiting for gerrit replication config map appears in gerrit pod. Sleeping for 90 seconds...")
	time.Sleep(90 * time.Second)
	
	err = reloadReplicationPlugin(codebaseSettings.GerritKeyPath, codebaseSettings.GerritHost,
		codebaseSettings.GerritSettings.SshPort)
	if err != nil {
		log.Printf("Unable to reload replication plugin: %v", err)
		return err
	}
	log.Printf("Replication configuration has been finished for app %v", codebaseSettings.Name)

	return nil
}

func reloadReplicationPlugin(keyPath string, host string, port int64) error {
	command := "gerrit plugin reload replication"

	cmd := &SSHCommand{
		Path:   command,
		Env:    []string{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	client, err := SshInit(keyPath, host, port)
	if err != nil {
		return err
	}

	_, err = client.RunCommand(cmd)
	if err != nil {
		return err
	}

	log.Printf("Gerrit replication plugin has been reloaded Host: %v Port: %v KeyPath: %v", host, port, keyPath)
	return nil
}
