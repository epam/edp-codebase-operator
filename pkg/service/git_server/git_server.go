package git_server

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	goGit "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
	"time"
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

	log.Info("Data from request is extracted", "host", gitSshData.Host, "port", gitSshData.Port)

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

func initAuth(data model.RepositoryData) (*goGit.PublicKeys, error) {
	log.Info("Initializing auth", "user", data.User)

	signer, err := ssh.ParsePrivateKey([]byte(data.Key))
	if err != nil {
		return nil, err
	}

	return &goGit.PublicKeys{
		User:   data.User,
		Signer: signer,
		HostKeyCallbackHelper: goGit.HostKeyCallbackHelper{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}, nil
}

func (s *GitServerService) CloneRepository(data model.RepositoryData) error {
	log.Info("Start cloning", "repository", data.RepositoryUrl)

	auth, err := initAuth(data)
	if err != nil {
		return err
	}

	_, err = git.PlainClone(data.FolderToClone, false, &git.CloneOptions{
		URL:  data.RepositoryUrl,
		Auth: auth,
	})
	if err != nil {
		return err
	}

	log.Info("End cloning", "repository", data.RepositoryUrl)

	return nil
}

func (s *GitServerService) CommitChanges(directoryToCommit string) error {
	log.Info("Start commiting changes", "directory", directoryToCommit)

	repository, err := git.PlainOpen(directoryToCommit)
	if err != nil {
		return err
	}

	worktree, err := repository.Worktree()

	_, err = worktree.Add(".")
	if err != nil {
		return err
	}

	array := strings.Split(directoryToCommit, "/")
	_, err = worktree.Commit(fmt.Sprintf("Add template for %v", array[len(array)-1]), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "admin",
			Email: "admin@epam-edp.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	log.Info("Changes has been commited", "directory", directoryToCommit)

	return nil
}

func (s *GitServerService) PushChanges(data model.RepositoryData, directoryToCommit string) error {
	log.Info("Start pushing changes", "directory", directoryToCommit)

	auth, err := initAuth(data)
	if err != nil {
		return err
	}

	r, err := git.PlainOpen(directoryToCommit)
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
		return err
	}

	log.Info("Changes has been pushed", "directory", directoryToCommit)

	return nil
}
