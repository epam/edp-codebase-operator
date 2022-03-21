package gitserver

import (
	"fmt"
	"io/ioutil"
	netHttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const tempDir = "/tmp"

type GitSshData struct {
	Host string
	User string
	Key  string
	Port int32
}

type Git interface {
	CommitChanges(directory, commitMsg string) error
	PushChanges(key, user, directory string, pushParams ...string) error
	CheckPermissions(repo string, user, pass *string) (accessible bool)
	CloneRepositoryBySsh(key, user, repoUrl, destination string, port int32) error
	CloneRepository(repo string, user *string, pass *string, destination string) error
	CreateRemoteBranch(key, user, path, name string) error
	CreateRemoteTag(key, user, path, branchName, name string) error
	Fetch(key, user, path, branchName string) error
	Checkout(user, pass *string, directory, branchName string, remote bool) error
	GetCurrentBranchName(directory string) (string, error)
	Init(directory string) error
	CheckoutRemoteBranchBySSH(key, user, gitPath, remoteBranchName string) error
	RemoveBranch(directory, branchName string) error
	RenameBranch(directory, currentName, newName string) error
	CreateChildBranch(directory, currentBranch, newBranch string) error
}

type Command interface {
	CombinedOutput() ([]byte, error)
}

type GitProvider struct {
	commandBuilder func(cmd string, params ...string) Command
}

func (gp *GitProvider) buildCommand(cmd string, params ...string) Command {
	if gp.commandBuilder == nil {
		gp.commandBuilder = func(cmd string, params ...string) Command {
			return exec.Command(cmd, params...)
		}
	}

	return gp.commandBuilder(cmd, params...)
}

var log = ctrl.Log.WithName("git-provider")

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
			Name:  "codebase",
			Email: "codebase@edp.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	log.Info("Changes have been commited", "directory", directory)
	return nil
}

func (gp GitProvider) RemoveBranch(directory, branchName string) error {
	cmd := gp.buildCommand("git", "--git-dir", fmt.Sprintf("%s/.git", directory), "branch", "-D", branchName)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to remove branch, err: %s", string(bts))
	}

	return nil
}

func (gp GitProvider) RenameBranch(directory, currentName, newName string) error {
	cmd := gp.buildCommand("git", "--git-dir", fmt.Sprintf("%s/.git", directory), "checkout",
		currentName)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to checkout branch, err: %s", string(bts))
	}

	cmd = gp.buildCommand("git", "--git-dir", fmt.Sprintf("%s/.git", directory), "branch", "-m",
		newName)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to rename branch, err: %s", string(bts))
	}

	return nil
}

func (gp GitProvider) CreateChildBranch(directory, currentBranch, newBranch string) error {
	cmd := gp.buildCommand("git", "--git-dir", fmt.Sprintf("%s/.git", directory), "checkout",
		currentBranch)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to checkout branch, err: %s", string(bts))
	}

	cmd = gp.buildCommand("git", "--git-dir", fmt.Sprintf("%s/.git", directory), "checkout", "-b",
		newBranch)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to rename branch, err: %s", string(bts))
	}

	return nil
}

func (GitProvider) PushChanges(key, user, directory string, pushParams ...string) error {
	log.Info("Start pushing changes", "directory", directory)
	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}
	defer os.Remove(keyPath)

	basePushParams := []string{"--git-dir", fmt.Sprintf("%s/.git", directory),
		"push", "origin"}
	basePushParams = append(basePushParams, pushParams...)

	pushCMD := exec.Command("git", basePushParams...)
	pushCMD.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no`, keyPath,
		user), "GIT_SSH_VARIANT=ssh"}
	pushCMD.Dir = directory
	log.Info("pushCMD:", "is: ", basePushParams)
	if bts, err := pushCMD.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to push changes, err: %s", string(bts))
	}

	log.Info("Changes has been pushed", "directory", directory)
	return nil
}

func (GitProvider) CheckPermissions(repo string, user, pass *string) (accessible bool) {
	log.Info("checking permissions", "user", user, "repository", repo)
	if user == nil || pass == nil {
		return true
	}

	r, _ := git.Init(memory.NewStorage(), nil)
	remote, _ := r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo},
	})
	rfs, err := remote.List(&git.ListOptions{
		Auth: &http.BasicAuth{
			Username: *user,
			Password: *pass,
		}})
	if err != nil {
		log.Error(err, fmt.Sprintf("User %v do not have access to %v repository", user, repo))
		return false
	}

	if len(rfs) == 0 {
		log.Error(errors.New("there are not refs in repository"), "no refs in repository")
		return false
	}

	return true
}

func (GitProvider) BareToNormal(path string) error {
	if err := os.MkdirAll(fmt.Sprintf("%s/.git", path), 0777); err != nil {
		return errors.Wrap(err, "unable to create .git folder")
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return errors.Wrap(err, "unable to list dir")
	}

	for _, f := range files {
		if f.Name() == ".git" {
			continue
		}

		if err := os.Rename(fmt.Sprintf("%s/%s", path, f.Name()),
			fmt.Sprintf("%s/.git/%s", path, f.Name())); err != nil {
			return errors.Wrap(err, "unable to rename file")
		}
	}

	gitDir := fmt.Sprintf("%s/.git", path)
	cmd := exec.Command("git", "--git-dir", gitDir, "config", "--local",
		"--bool", "core.bare", "false")
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, string(bts))
	}

	cmd = exec.Command("git", "--git-dir", gitDir, "config", "--local",
		"--bool", "remote.origin.mirror", "false")
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, string(bts))
	}

	cmd = exec.Command("git", "--git-dir", gitDir, "reset", "--hard")
	cmd.Dir = path
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, string(bts))
	}

	return nil
}

func (g GitProvider) CloneRepositoryBySsh(key, user, repoUrl, destination string, port int32) error {
	log.Info("Start cloning", "repository", repoUrl)
	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}
	defer os.Remove(keyPath)

	cloneCMD := exec.Command("git", "clone", "--mirror", "--depth", "1", repoUrl, destination)
	cloneCMD.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no -p %d`,
		keyPath, user, port), "GIT_SSH_VARIANT=ssh"}
	if bytes, err := cloneCMD.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to clone repo by ssh, err: %s", string(bytes))
	}

	fetchCMD := exec.Command("git", "--git-dir", destination, "fetch",
		"--unshallow")
	fetchCMD.Env = cloneCMD.Env
	bts, err := fetchCMD.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "unable to fetch unshallow repo: %s", string(bts))
	}
	log.Info("unshallow", "out", string(bts))

	if err := g.BareToNormal(destination); err != nil {
		return errors.Wrap(err, "unable to covert bare repo to normal")
	}

	fetchCMD = exec.Command("git", "--git-dir", path.Join(destination, ".git"), "pull", "origin", "master",
		"--unshallow", "--no-rebase")
	fetchCMD.Env = cloneCMD.Env
	bts, err = fetchCMD.CombinedOutput()
	if err != nil && !strings.Contains(string(bts), "does not make sense") {
		return errors.Wrapf(err, "unable to pull unshallow repo: %s", string(bts))
	}
	log.Info("unshallow", "out", string(bts))

	log.Info("End cloning", "repository", repoUrl)
	return nil
}

func (g GitProvider) CloneRepository(repo string, user *string, pass *string, destination string) error {
	log.Info("Start cloning", "repository", repo)

	if user != nil && pass != nil {
		u, err := url.Parse(repo)
		if err != nil {
			return errors.Wrap(err, "unable to parse repo url")
		}
		u.User = url.UserPassword(*user, *pass)
		repo = fmt.Sprint(u)
	} else {
		rsp, err := netHttp.Get(repo)
		if err != nil {
			return errors.Wrap(err, "unable to get repo")
		}
		if rsp.StatusCode >= 400 {
			return errors.Wrapf(err, "repo access denied, response code: %d", rsp.StatusCode)
		}
	}

	cloneCMD := exec.Command("git", "clone", "--mirror", "--depth", "1", repo, destination)

	if bts, err := cloneCMD.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to clone repo: %s", string(bts))
	}

	fetchCMD := exec.Command("git", "--git-dir", destination, "fetch",
		"--unshallow")
	bts, err := fetchCMD.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "unable to fetch unshallow repo: %s", string(bts))
	}
	log.Info("unshallow", "out", string(bts))

	if err := g.BareToNormal(destination); err != nil {
		return errors.Wrap(err, "unable to covert bare repo to normal")
	}

	fetchCMD = exec.Command("git", "--git-dir", path.Join(destination, ".git"), "pull", "origin", "master",
		"--unshallow", "--no-rebase")
	bts, err = fetchCMD.CombinedOutput()
	if err != nil && !strings.Contains(string(bts), "does not make sense") {
		return errors.Wrapf(err, "unable to pull unshallow repo: %s", string(bts))
	}
	log.Info("unshallow", "out", string(bts))

	log.Info("End cloning", "repository", repo)
	return nil
}

func initAuth(key, user string) (string, error) {
	log.Info("Initializing auth", "user", user)
	keyFile, err := os.Create(fmt.Sprintf("%s/sshkey_%d", tempDir, time.Now().Unix()))
	if err != nil {
		return "", errors.Wrap(err, "unable to create temp file for ssh key")
	}
	keyFileInfo, _ := keyFile.Stat()
	keyFilePath := fmt.Sprintf("%s/%s", tempDir, keyFileInfo.Name())

	if _, err = keyFile.WriteString(key); err != nil {
		return "", errors.Wrap(err, "unable to write ssh key")
	}

	if err = keyFile.Close(); err != nil {
		return "", errors.Wrap(err, "unable to close file")
	}

	if err := os.Chmod(keyFilePath, 0400); err != nil {
		return "", errors.Wrap(err, "unable to chmod ssh key file")
	}

	return keyFilePath, nil
}

func checkConnectionToGitServer(c client.Client, gitServer model.GitServer) (bool, error) {
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
	sshClient, err := sshInitFromSecret(data, log)
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

func sshInitFromSecret(data GitSshData, logger logr.Logger) (gerrit.SSHClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: data.User,
		Auth: []ssh.AuthMethod{
			publicKey(data.Key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	cl := gerrit.SSHClient{
		Config: sshConfig,
		Host:   data.Host,
		Port:   data.Port,
	}

	logger.Info("SSH Client has been initialized", "host", data.Host, "port", data.Port)

	return cl, nil
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

	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}
	defer os.Remove(keyPath)

	cmd := exec.Command("git", "--git-dir", fmt.Sprintf("%s/.git", path), "fetch",
		fmt.Sprintf("refs/heads/%v:refs/heads/%v", branchName, branchName))
	cmd.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no`, keyPath, user),
		"GIT_SSH_VARIANT=ssh"}
	cmd.Dir = path
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to push changes, err: %s", string(bts))
	}

	log.Info("end fetching data", "name", branchName)
	return nil
}

func (gp GitProvider) Checkout(user, pass *string, directory, branchName string, remote bool) error {
	log.Info("trying to checkout to branch", "name", branchName)
	r, err := git.PlainOpen(directory)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	createBranchOrNot := true
	if remote {
		gfo := &git.FetchOptions{RefSpecs: []config.RefSpec{"refs/*:refs/*"}}
		if user != nil && pass != nil {
			gfo = &git.FetchOptions{
				RefSpecs: []config.RefSpec{"refs/*:refs/*"},
				Auth: &http.BasicAuth{
					Username: *user,
					Password: *pass,
				},
			}
		}

		err = r.Fetch(gfo)
		if err != nil {
			if err.Error() != "already up-to-date" {
				return errors.Wrapf(err, "Unable to fetch")
			}
		}

		createBranchOrNot, err = checkBranchExistence(user, pass, branchName, *r)
		if err != nil {
			return err
		}
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)),
		Force:  true,
		Create: createBranchOrNot,
	})
	if err != nil {
		return err
	}
	return nil
}

func (gp GitProvider) GetCurrentBranchName(directory string) (string, error) {
	log.Info("trying to get current git branch")
	r, err := git.PlainOpen(directory)
	if err != nil {
		return "", err
	}

	ref, err := r.Head()
	if err != nil {
		return "", err
	}
	branchName := strings.ReplaceAll(ref.Name().String(), "refs/heads/", "")
	return branchName, nil
}

func (gp GitProvider) Init(directory string) error {
	log.Info("start creating git repository")
	_, err := git.PlainInit(directory, false)
	if err != nil {
		return err
	}
	log.Info("git repository has been created")
	return nil
}

func checkBranchExistence(user, pass *string, branchName string, r git.Repository) (bool, error) {
	log.Info("checking if branch exist", "branchName", branchName)
	remote, err := r.Remote("origin")
	if err != nil {
		return false, err
	}
	glo := &git.ListOptions{}
	if user != nil && pass != nil {
		glo = &git.ListOptions{Auth: &http.BasicAuth{
			Username: *user,
			Password: *pass,
		},
		}
	}

	refList, err := remote.List(glo)
	if err != nil {
		return false, err
	}
	existBranchOrNot := true
	refPrefix := "refs/heads/"
	for _, ref := range refList {
		refName := ref.Name().String()
		if !strings.HasPrefix(refName, refPrefix) {
			continue
		}
		b := refName[len(refPrefix):]
		if b == branchName {
			existBranchOrNot = false
			break
		}
	}
	log.Info("branch existence status", "branchName", branchName, "existBranchOrNot", existBranchOrNot)
	return existBranchOrNot, nil
}

func (gp GitProvider) CheckoutRemoteBranchBySSH(key, user, gitPath, remoteBranchName string) error {
	log.Info("start checkout to", "branch", remoteBranchName)

	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}
	defer os.Remove(keyPath)

	// running git fetch --update-head-ok
	cmdFetch := exec.Command("git", "--git-dir", path.Join(gitPath, ".git"), "fetch", "--update-head-ok")
	cmdFetch.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no`, keyPath, user),
		"GIT_SSH_VARIANT=ssh"}
	cmdFetch.Dir = gitPath
	if bts, err := cmdFetch.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to fetch branches, err: %s", string(bts))
	}

	// here we expect that remote branch exists otherwise return error
	// git checkout -b remoteBranchName remoteBranchName
	cmdCheckout := exec.Command("git", "--git-dir", path.Join(gitPath, ".git"), "checkout", remoteBranchName)
	cmdCheckout.Dir = gitPath
	if bts, err := cmdCheckout.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to checkout to branch, err: %s", string(bts))
	}

	log.Info("end checkout to", "branch", remoteBranchName)
	return nil
}
