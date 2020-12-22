package gitserver

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	goGit "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	v1 "k8s.io/api/core/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"time"
)

type GitSshData struct {
	Host string
	User string
	Key  string
	Port int32
}

type Git interface {
	CommitChanges(directory, commitMsg string) error
	PushChanges(key, user, directory string) error
	CheckPermissions(repo string, user string, pass string) (accessible bool)
	CloneRepositoryBySsh(key, user, repoUrl, destination string) error
	CloneRepository(repo, user, pass, destination string) error
	CreateRemoteBranch(key, user, path, name string) error
	CreateRemoteTag(key, user, path, branchName, name string) error
	Fetch(key, user, path, branchName string) error
	Checkout(directory, branchName string) error
	CreateLocalBranch(path, name string) error
}

type GitProvider struct {
}

func (gp GitProvider) CreateRemoteBranch(key, user, path, name string) error {
	log.Info("start creating remote branch", "name", name)
	r, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	branches, err := r.Branches()
	if err != nil {
		return err
	}

	exists, err := isBranchExists(name, branches)
	if err != nil {
		return err
	}

	if exists {
		log.Info("branch already exists. skip creating", "name", name)
		return nil
	}

	ref, err := r.Head()
	if err != nil {
		return err
	}

	newRef := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/heads/%v", name), ref.Hash().String())
	if err := r.Storer.SetReference(newRef); err != nil {
		return err
	}

	if err := gp.PushChanges(key, user, path); err != nil {
		return err
	}
	log.Info("branch has been created", "name", name)
	return nil
}

func isBranchExists(name string, branches storer.ReferenceIter) (bool, error) {
	for {
		b, err := branches.Next()
		if err != nil {
			if err.Error() == "EOF" {
				return false, nil
			}
			return false, err
		}
		if b.Name().Short() == name {
			return true, nil
		}
	}
}

func (GitProvider) CommitChanges(directory, commitMsg string) error {
	log.Info("Start commiting changes", "directory", directory)
	r, err := git.PlainOpen(directory)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Add(".")
	if err != nil {
		return err
	}

	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "admin",
			Email: "admin@epam-edp.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	log.Info("Changes have been commited", "directory", directory)
	return nil
}

func (GitProvider) PushChanges(key, user, directory string) error {
	log.Info("Start pushing changes", "directory", directory)
	auth, err := initAuth(key, user)
	if err != nil {
		return err
	}

	r, err := git.PlainOpen(directory)
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
	log.Info("Changes has been pushed", "directory", directory)
	return nil
}

func (GitProvider) CheckPermissions(repo string, user string, pass string) (accessible bool) {
	log.Info("checking permissions", "user", user, "repository", repo)
	r, _ := git.Init(memory.NewStorage(), nil)
	remote, _ := r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo},
	})
	rfs, err := remote.List(&git.ListOptions{
		Auth: &http.BasicAuth{
			Username: user,
			Password: pass,
		}})
	if err != nil {
		log.Error(err, fmt.Sprintf("User %v do not have access to %v repository", user, repo))
		return false
	}
	return len(rfs) != 0
}

func (GitProvider) CloneRepositoryBySsh(key, user, repoUrl, destination string) error {
	log.Info("Start cloning", "repository", repoUrl)
	auth, err := initAuth(key, user)
	if err != nil {
		return err
	}

	_, err = git.PlainClone(destination, false, &git.CloneOptions{
		URL:  repoUrl,
		Auth: auth,
	})
	if err != nil {
		return err
	}
	log.Info("End cloning", "repository", repoUrl)
	return nil
}

func (GitProvider) CloneRepository(repo, user, pass, destination string) error {
	log.Info("Start cloning", "repository", repo)
	_, err := git.PlainClone(destination, false, &git.CloneOptions{
		URL: repo,
		Auth: &http.BasicAuth{
			Username: user,
			Password: pass,
		}})
	if err != nil {
		return err
	}
	log.Info("End cloning", "repository", repo)
	return nil
}

func initAuth(key, user string) (*goGit.PublicKeys, error) {
	log.Info("Initializing auth", "user", user)
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return nil, err
	}

	return &goGit.PublicKeys{
		User:   user,
		Signer: signer,
		HostKeyCallbackHelper: goGit.HostKeyCallbackHelper{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}, nil
}

func checkConnectionToGitServer(c coreV1Client.CoreV1Client, gitServer model.GitServer) (bool, error) {
	log.Info("Start CheckConnectionToGitServer method", "Git host", gitServer.GitHost)

	sshSecret, err := util.GetSecret(c, gitServer.NameSshKeySecret, gitServer.Namespace)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("an error has occurred  while getting %v secret", gitServer.NameSshKeySecret))
	}

	gitSshData := extractSshData(gitServer, sshSecret)

	log.Info("Data from request is extracted", "host", gitSshData.Host, "port", gitSshData.Port)

	a := isGitServerAccessible(gitSshData)
	log.Info("Git server", "accessible", a)
	return a, nil
}

func isGitServerAccessible(data GitSshData) bool {
	log.Info("Start executing IsGitServerAccessible method to check connection to server", "host", data.Host)
	sshClient, err := sshInitFromSecret(data)
	if err != nil {
		log.Info(fmt.Sprintf("An error has occurred while initing SSH client. Check data in Git Server resource and secret: %v", err))
		return false
	}

	var s *ssh.Session
	var c *ssh.Client
	if s, c, err = sshClient.NewSession(); err != nil {
		log.Info(fmt.Sprintf("An error has occurred while connecting to server. Check data in Git Server resource and secret: %v", err))
		return false
	}
	defer s.Close()
	defer c.Close()

	return s != nil && c != nil
}

func extractSshData(gitServer model.GitServer, secret *v1.Secret) GitSshData {
	return GitSshData{
		Host: gitServer.GitHost,
		User: gitServer.GitUser,
		Key:  string(secret.Data[util.PrivateSShKeyName]),
		Port: gitServer.SshPort,
	}
}

func sshInitFromSecret(data GitSshData) (gerrit.SSHClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: data.User,
		Auth: []ssh.AuthMethod{
			publicKey(data.Key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client := &gerrit.SSHClient{
		Config: sshConfig,
		Host:   data.Host,
		Port:   data.Port,
	}
	log.Info("SSH Client has been initialized: Host: %v Port: %v", data.Host, data.Port)
	return *client, nil
}

func publicKey(key string) ssh.AuthMethod {
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}

func (gp GitProvider) CreateRemoteTag(key, user, path, branchName, name string) error {
	log.Info("start creating remote tag", "name", name)
	r, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	tags, err := r.Tags()
	if err != nil {
		return err
	}

	exists, err := isTagExists(name, tags)
	if err != nil {
		return err
	}

	if exists {
		log.Info("tag already exists. skip creating", "name", name)
		return nil
	}

	ref, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%v", branchName)), false)
	if err != nil {
		return err
	}

	newRef := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/tags/%v", name), ref.Hash().String())
	if err := r.Storer.SetReference(newRef); err != nil {
		return err
	}

	if err := gp.PushChanges(key, user, path); err != nil {
		return err
	}
	log.Info("tag has been created", "name", name)
	return nil
}

func isTagExists(name string, tags storer.ReferenceIter) (bool, error) {
	for {
		t, err := tags.Next()
		if err != nil {
			if err.Error() == "EOF" {
				return false, nil
			}
			return false, err
		}
		if t.Name().Short() == name {
			return true, nil
		}
	}
}

func (gp GitProvider) Fetch(key, user, path, branchName string) error {
	log.Info("start fetching data", "name", branchName)
	r, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	auth, err := initAuth(key, user)
	if err != nil {
		return err
	}

	opts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%v:refs/heads/%v", branchName, branchName)),
		},
		Auth: auth,
	}

	if err := r.Fetch(opts); err != nil {
		if err.Error() == "already up-to-date" {
			log.Info("repo is already up-to-date")
			return nil
		}
		return err
	}
	log.Info("end fetching data", "name", branchName)
	return nil
}

func (gp GitProvider) Checkout(directory, branchName string) error {
	log.Info("start checkout branch", "name", branchName)
	r, err := git.PlainOpen(directory)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	err = w.Checkout(&git.CheckoutOptions{Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%v", branchName))})
	if err != nil{
		return err
	}
	return nil
}

func (gp GitProvider) CreateLocalBranch(path, name string) error {
	log.Info("start creating local branch", "name", name)
	r, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	branches, err := r.Branches()
	if err != nil {
		return err
	}

	exists, err := isBranchExists(name, branches)
	if err != nil {
		return err
	}

	if exists {
		log.Info("branch already exists. skip creating", "name", name)
		return nil
	}

	ref, err := r.Head()
	if err != nil {
		return err
	}

	newRef := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/heads/%v", name), ref.Hash().String())
	if err := r.Storer.SetReference(newRef); err != nil {
		return err
	}
	log.Info("local branch has been created", "name", name)
	return nil
}
