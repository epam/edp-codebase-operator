package gerrit

import (
	"business-app-handler-controller/models"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"bytes"
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

func (client *SSHClient) RunCommand(cmd *SSHCommand) ([]byte, error) {
	var session *ssh.Session
	var err error

	if session, err = client.newSession(); err != nil {
		return nil, err
	}
	defer session.Close()

	commandOutput, err := session.Output(cmd.Path)
	if err != nil {
		return nil, err
	}

	return commandOutput, err
}

func (client *SSHClient) newSession() (*ssh.Session, error) {
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", client.Host, client.Port), client.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %s", err)
	}

	return session, nil
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

	client.RunCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

func AddRemoteLinkToGerrit(repoPath string, host string, port int64, appName string) error {
	remoteUrl := fmt.Sprintf("ssh://project-creator@%v:%v/%v", host, string(port), appName)
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

	signer, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return &sshgit.PublicKeys{
		User:   "project-creator",
		Signer: signer,
		HostKeyCallbackHelper: sshgit.HostKeyCallbackHelper{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}, nil
}

func PushToGerrit(repoPath string, keyPath string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	auth, err := Auth(keyPath)
	if err != nil {
		return err
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"refs/heads/*:refs/heads/*",
			"refs/tags/*:refs/tags/*",
		},
		Auth: auth,
	})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Pushed to gerrit repo %v", repoPath)
	return nil
}

func SetupProjectReplication(appSettings models.AppSettings, clientSet ClientSet.ClientSet) error {
	var renderedTemplate bytes.Buffer
	tmpl, err := template.New("replication-conf.tmpl").
		ParseFiles("/usr/local/bin/templates/gerrit/replication-conf.tmpl")
	if err != nil {
		return err
	}
	err = tmpl.Execute(&renderedTemplate, appSettings)
	if err != nil {
		log.Printf("Unable to render replication config: %v", err)
		return err
	}

	gerritSettings, err := clientSet.CoreClient.ConfigMaps(appSettings.CicdNamespace).Get("gerrit", metav1.GetOptions{})
	replicaConfig := gerritSettings.Data["replication.config"]
	gerritSettings.Data["replication.config"] = replicaConfig + renderedTemplate.String()
	result, err := clientSet.CoreClient.ConfigMaps(appSettings.CicdNamespace).Update(gerritSettings)
	if err != nil {
		log.Printf("Unable to update config map with replication config: %v", err)
		return err
	}
	log.Println(result)
	err = reloadReplicationPlugin(appSettings.GerritKeyPath, appSettings.GerritHost,
		appSettings.GerritSettings.SshPort)
	if err != nil {
		log.Printf("Unable to reload replication plugin: %v", err)
		return err
	}
	log.Printf("Replication configuration has been finished for app %v", appSettings.Name)

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

	client.RunCommand(cmd)
	if err != nil {
		return err
	}

	log.Printf("Gerrit replication plugin has been reloaded Host: %v Port: %v KeyPath: %v", host, port, keyPath)
	return nil
}
