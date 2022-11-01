package gitserver

import (
	"fmt"
	netHttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

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

	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	gitCMD          = "git"
	gitDirName      = ".git"
	gitDirArg       = "--git-dir"
	gitBranchArg    = "branch"
	gitCheckoutArg  = "checkout"
	getFetchArg     = "fetch"
	gitOriginArg    = "origin"
	gitUnshallowArg = "--unshallow"

	gitSshVariantEnv = "GIT_SSH_VARIANT=ssh"
)

const defaultSshPort = 22

const (
	logBranchNameKey = "branchName"
	logDirectoryKey  = "directory"
	logRepositoryKey = "repository"
	logOutKey        = "out"

	errPlainOpenTmpl    = "failed to open git directory %q: %w"
	errRemoveSHHKeyFile = "unable to remove key file"
)

type GitSshData struct {
	Host string
	User string
	Key  string
	Port int32
}

type Git interface {
	CommitChanges(directory, commitMsg string) error
	PushChanges(key, user, directory string, port int32, pushParams ...string) error
	CheckPermissions(repo string, user, pass *string) (accessible bool)
	CloneRepositoryBySsh(key, user, repoUrl, destination string, port int32) error
	CloneRepository(repo string, user *string, pass *string, destination string) error
	CreateRemoteBranch(key, user, path, name, fromcommit string, port int32) error
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

func (gp *GitProvider) CreateRemoteBranch(key, user, p, name, fromcommit string, port int32) error {
	log.Info("start creating remote branch", logBranchNameKey, name)

	r, err := git.PlainOpen(p)
	if err != nil {
		return fmt.Errorf(errPlainOpenTmpl, p, err)
	}

	branches, err := r.Branches()
	if err != nil {
		return fmt.Errorf("failed to get branches iterator: %w", err)
	}

	exists, err := isBranchExists(name, branches)
	if err != nil {
		return err
	}

	if exists {
		log.Info("branch already exists. skip creating", logBranchNameKey, name)
		return nil
	}

	ref, err := r.Head()
	if err != nil {
		return fmt.Errorf("failed to get git HEAD reference: %w", err)
	}

	if fromcommit != "" {
		ref = plumbing.NewReferenceFromStrings(name, fromcommit)

		if err != nil {
			return fmt.Errorf("failed to create a reference from name: %w", err)
		}
	}

	newRef := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/heads/%v", name), ref.Hash().String())

	err = r.Storer.SetReference(newRef)
	if err != nil {
		return fmt.Errorf("failed to set refference: %w", err)
	}

	err = gp.PushChanges(key, user, p, port, "--all")
	if err != nil {
		return err
	}

	log.Info("branch has been created", logBranchNameKey, name)

	return nil
}

func (*GitProvider) CommitChanges(directory, commitMsg string) error {
	log.Info("Start committing changes", logDirectoryKey, directory)

	r, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf(errPlainOpenTmpl, directory, err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get git worktree: %w", err)
	}

	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add files to the index: %w", err)
	}

	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "codebase",
			Email: "codebase@edp.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Info("Changes have been committed", logDirectoryKey, directory)

	return nil
}

func (gp *GitProvider) RemoveBranch(directory, branchName string) error {
	gitDir := path.Join(directory, gitDirName)
	cmd := gp.buildCommand(gitCMD, gitDirArg, gitDir, gitBranchArg, "-D", branchName)

	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to remove branch, err: %s", string(bts))
	}

	return nil
}

func (gp *GitProvider) RenameBranch(directory, currentName, newName string) error {
	gitDir := path.Join(directory, gitDirName)
	cmd := gp.buildCommand(gitCMD, gitDirArg, gitDir, gitCheckoutArg, currentName)

	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to checkout branch, err: %s", string(bts))
	}

	cmd = gp.buildCommand(gitCMD, gitDirArg, gitDir, gitBranchArg, "-m", newName)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to rename branch, err: %s", string(bts))
	}

	return nil
}

func (gp *GitProvider) CreateChildBranch(directory, currentBranch, newBranch string) error {
	gitDir := path.Join(directory, gitDirName)
	cmd := gp.buildCommand(gitCMD, gitDirArg, gitDir, gitCheckoutArg, currentBranch)

	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to checkout branch, err: %s", string(bts))
	}

	cmd = gp.buildCommand(gitCMD, gitDirArg, gitDir, gitCheckoutArg, "-b", newBranch)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to rename branch, err: %s", string(bts))
	}

	return nil
}

func (*GitProvider) PushChanges(key, user, directory string, port int32, pushParams ...string) (err error) {
	log.Info("Start pushing changes", logDirectoryKey, directory)

	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if cleanKeyErr := os.Remove(keyPath); err != nil {
			log.Error(cleanKeyErr, errRemoveSHHKeyFile)
		}
	}()

	gitDir := path.Join(directory, gitDirName)
	basePushParams := []string{gitDirArg, gitDir, "push", gitOriginArg}
	basePushParams = append(basePushParams, pushParams...)

	pushCMD := exec.Command(gitCMD, basePushParams...)
	pushCMD.Env = []string{
		fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no -p %d`, keyPath,
			user, port),
		gitSshVariantEnv,
	}
	pushCMD.Dir = directory

	log.Info("pushCMD:", "is: ", basePushParams)

	if bts, err := pushCMD.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to push changes, err: %s", string(bts))
	}

	log.Info("Changes has been pushed", logDirectoryKey, directory)

	return
}

func (*GitProvider) CheckPermissions(repo string, user, pass *string) (accessible bool) {
	log.Info("checking permissions", "user", user, logRepositoryKey, repo)

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

func (*GitProvider) BareToNormal(p string) error {
	const readWriteExecutePermBits = 0o777

	gitDir := path.Join(p, gitDirName)

	if err := os.MkdirAll(gitDir, readWriteExecutePermBits); err != nil {
		return errors.Wrap(err, "unable to create .git folder")
	}

	files, err := os.ReadDir(p)
	if err != nil {
		return errors.Wrap(err, "unable to list dir")
	}

	for _, f := range files {
		if f.Name() == gitDirName {
			continue
		}

		oldPath := path.Join(p, f.Name())
		newPath := path.Join(p, gitDirName, f.Name())

		if err := os.Rename(oldPath, newPath); err != nil {
			return errors.Wrap(err, "unable to rename file")
		}
	}

	cmd := exec.Command(gitCMD, gitDirArg, gitDir, "config", "--local",
		"--bool", "core.bare", "false")
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, string(bts))
	}

	cmd = exec.Command(gitCMD, gitDirArg, gitDir, "config", "--local",
		"--bool", "remote.origin.mirror", "false")
	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, string(bts))
	}

	cmd = exec.Command(gitCMD, gitDirArg, gitDir, "reset", "--hard")
	cmd.Dir = p

	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, string(bts))
	}

	return nil
}

func (gp *GitProvider) CloneRepositoryBySsh(key, user, repoUrl, destination string, port int32) (err error) {
	log.Info("Start cloning", logRepositoryKey, repoUrl)

	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if cleanKeyErr := os.Remove(keyPath); err != nil {
			log.Error(cleanKeyErr, errRemoveSHHKeyFile)
		}
	}()

	cloneCMD := exec.Command(gitCMD, "clone", "--mirror", "--depth", "1", repoUrl, destination)
	cloneCMD.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no -p %d`,
		keyPath, user, port), gitSshVariantEnv}

	bytes, err := cloneCMD.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "unable to clone repo by ssh, err: %s", string(bytes))
	}

	fetchCMD := exec.Command(gitCMD, gitDirArg, destination, getFetchArg, gitUnshallowArg)
	fetchCMD.Env = cloneCMD.Env

	bts, err := fetchCMD.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "unable to fetch unshallow repo: %s", string(bts))
	}

	log.Info("Result of `git fetch unshallow` command", logOutKey, string(bts))

	err = gp.BareToNormal(destination)
	if err != nil {
		return errors.Wrap(err, "unable to covert bare repo to normal")
	}

	gitDir := path.Join(destination, ".git")
	fetchCMD = exec.Command(gitCMD, gitDirArg, gitDir, "pull", gitOriginArg, gitUnshallowArg, "--no-rebase")
	fetchCMD.Env = cloneCMD.Env
	bts, err = fetchCMD.CombinedOutput()

	if err != nil && !strings.Contains(string(bts), "does not make sense") {
		return errors.Wrapf(err, "unable to pull unshallow repo: %s", string(bts))
	}

	log.Info("Result of `git pull unshallow` command", logOutKey, string(bts))

	log.Info("End cloning", logRepositoryKey, repoUrl)

	return
}

func (gp *GitProvider) CloneRepository(repo string, user, pass *string, destination string) error {
	log.Info("Start cloning", logRepositoryKey, repo)

	const httpClientErrors = 400

	if user != nil && pass != nil {
		u, err := url.Parse(repo)
		if err != nil {
			return errors.Wrap(err, "unable to parse repo url")
		}

		u.User = url.UserPassword(*user, *pass)
		repo = u.String()
	} else {
		rsp, err := netHttp.Get(repo)
		if err != nil {
			return errors.Wrap(err, "unable to get repo")
		}
		if rsp.StatusCode >= httpClientErrors {
			return errors.Wrapf(err, "repo access denied, response code: %d", rsp.StatusCode)
		}
	}

	cloneCMD := exec.Command(gitCMD, "clone", "--mirror", "--depth", "1", repo, destination)

	if bts, err := cloneCMD.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to clone repo: %s", string(bts))
	}

	fetchCMD := exec.Command(gitCMD, gitDirArg, destination, getFetchArg, gitUnshallowArg)

	bts, err := fetchCMD.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "unable to fetch unshallow repo: %s", string(bts))
	}

	log.Info("Result of `git fetch unshallow` command", logOutKey, string(bts))

	err = gp.BareToNormal(destination)
	if err != nil {
		return errors.Wrap(err, "unable to covert bare repo to normal")
	}

	gitDir := path.Join(destination, gitDirName)
	fetchCMD = exec.Command(gitCMD, gitDirArg, gitDir, "pull", gitOriginArg, gitUnshallowArg, "--no-rebase")

	bts, err = fetchCMD.CombinedOutput()
	if err != nil && !strings.Contains(string(bts), "does not make sense") {
		return errors.Wrapf(err, "unable to pull unshallow repo: %s", string(bts))
	}

	log.Info("Result of `git pull unshallow` command", logOutKey, string(bts))
	log.Info("End cloning", logRepositoryKey, repo)

	return nil
}

func (gp *GitProvider) CreateRemoteTag(key, user, p, branchName, name string) error {
	log.Info("start creating remote tag", "tagName", name)

	r, err := git.PlainOpen(p)
	if err != nil {
		return fmt.Errorf(errPlainOpenTmpl, p, err)
	}

	tags, err := r.Tags()
	if err != nil {
		return fmt.Errorf("failed to get git tags: %w", err)
	}

	exists, err := isTagExists(name, tags)
	if err != nil {
		return err
	}

	if exists {
		log.Info("tag already exists. skip creating", "tagName", name)
		return nil
	}

	ref, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%v", branchName)), false)
	if err != nil {
		return fmt.Errorf("failed to get reference: %w", err)
	}

	newRef := plumbing.NewReferenceFromStrings(fmt.Sprintf("refs/tags/%v", name), ref.Hash().String())

	err = r.Storer.SetReference(newRef)
	if err != nil {
		return fmt.Errorf("failed to set refference: %w", err)
	}

	err = gp.PushChanges(key, user, p, defaultSshPort)
	if err != nil {
		return err
	}

	log.Info("tag has been created", "tagName", name)

	return nil
}

func (*GitProvider) Fetch(key, user, workDir, branchName string) (err error) {
	log.Info("start fetching data", logBranchNameKey, branchName)

	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if cleanKeyErr := os.Remove(keyPath); err != nil {
			log.Error(cleanKeyErr, errRemoveSHHKeyFile)
		}
	}()

	gitDir := path.Join(workDir, gitDirName)
	cmd := exec.Command(gitCMD, gitDirArg, gitDir, getFetchArg, fmt.Sprintf("refs/heads/%v:refs/heads/%v", branchName, branchName))
	cmd.Env = []string{
		fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no`, keyPath, user),
		gitSshVariantEnv,
	}
	cmd.Dir = workDir

	if bts, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to push changes, err: %s", string(bts))
	}

	log.Info("end fetching data", logBranchNameKey, branchName)

	return
}

func (*GitProvider) Checkout(user, pass *string, directory, branchName string, remote bool) error {
	log.Info("trying to checkout to branch", logBranchNameKey, branchName)

	r, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf(errPlainOpenTmpl, directory, err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get git worktree: %w", err)
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
		return fmt.Errorf("failed to checkout git branch: %w", err)
	}

	return nil
}

func (*GitProvider) GetCurrentBranchName(directory string) (string, error) {
	log.Info("trying to get current git branch")

	r, err := git.PlainOpen(directory)
	if err != nil {
		return "", fmt.Errorf(errPlainOpenTmpl, directory, err)
	}

	ref, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD refference: %w", err)
	}

	branchName := strings.ReplaceAll(ref.Name().String(), "refs/heads/", "")

	return branchName, nil
}

func (*GitProvider) Init(directory string) error {
	log.Info("start creating git repository")

	_, err := git.PlainInit(directory, false)
	if err != nil {
		return fmt.Errorf("failed to init Git repository: %w", err)
	}

	log.Info("git repository has been created")

	return nil
}

func (*GitProvider) CheckoutRemoteBranchBySSH(key, user, gitPath, remoteBranchName string) (err error) {
	log.Info("start checkout to", "branch", remoteBranchName)

	keyPath, err := initAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if cleanKeyErr := os.Remove(keyPath); err != nil {
			log.Error(cleanKeyErr, errRemoveSHHKeyFile)
		}
	}()

	gitDir := path.Join(gitPath, ".git")

	// running git fetch --update-head-ok
	cmdFetch := exec.Command(gitCMD, gitDirArg, gitDir, getFetchArg, "--update-head-ok")
	cmdFetch.Env = []string{
		fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no`, keyPath, user),
		gitSshVariantEnv,
	}
	cmdFetch.Dir = gitPath

	if bts, err := cmdFetch.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to fetch branches, err: %s", string(bts))
	}

	// here we expect that remote branch exists otherwise return error
	// git checkout -b remoteBranchName remoteBranchName
	cmdCheckout := exec.Command(gitCMD, gitDirArg, gitDir, gitCheckoutArg, remoteBranchName)
	cmdCheckout.Dir = gitPath

	if bts, err := cmdCheckout.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "unable to checkout to branch, err: %s", string(bts))
	}

	log.Info("end checkout to", "branch", remoteBranchName)

	return nil
}

func isBranchExists(name string, branches storer.ReferenceIter) (bool, error) {
	for {
		b, err := branches.Next()
		if err != nil {
			if err.Error() == "EOF" {
				return false, nil
			}

			return false, fmt.Errorf("failed to get next branch iterator: %w", err)
		}

		if b.Name().Short() == name {
			return true, nil
		}
	}
}

func initAuth(key, user string) (string, error) {
	log.Info("Initializing auth", "user", user)

	keyFile, err := os.Create(fmt.Sprintf("%s/sshkey_%d", os.TempDir(), time.Now().Unix()))
	if err != nil {
		return "", errors.Wrap(err, "unable to create temp file for ssh key")
	}

	keyFileInfo, _ := keyFile.Stat()
	keyFilePath := path.Join(os.TempDir(), keyFileInfo.Name())

	if _, err = keyFile.WriteString(key); err != nil {
		return "", errors.Wrap(err, "unable to write ssh key")
	}

	if err = keyFile.Close(); err != nil {
		return "", errors.Wrap(err, "unable to close file")
	}

	const readOnlyPermBits = 0o400

	if err := os.Chmod(keyFilePath, readOnlyPermBits); err != nil {
		return "", errors.Wrap(err, "unable to chmod ssh key file")
	}

	return keyFilePath, nil
}

func checkConnectionToGitServer(c client.Client, gitServer *model.GitServer) (bool, error) {
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

	var (
		s *ssh.Session
		c *ssh.Client
	)

	if s, c, err = sshClient.NewSession(); err != nil {
		log.Info(fmt.Sprintf("An error has occurred while connecting to server. Check data in Git Server resource and secret: %v", err))
		return false
	}

	defer util.CloseWithLogOnErr(log, s, "failed to close ssh client session")
	defer util.CloseWithLogOnErr(log, c, "failed to close ssh client connection")

	return s != nil && c != nil
}

func extractSshData(gitServer *model.GitServer, secret *v1.Secret) GitSshData {
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

func isTagExists(name string, tags storer.ReferenceIter) (bool, error) {
	for {
		t, err := tags.Next()
		if err != nil {
			if err.Error() == "EOF" {
				return false, nil
			}

			return false, fmt.Errorf("failed to get next reference iterator: %w", err)
		}

		if t.Name().Short() == name {
			return true, nil
		}
	}
}

func checkBranchExistence(user, pass *string, branchName string, r git.Repository) (bool, error) {
	log.Info("checking if branch exist", logBranchNameKey, branchName)

	remote, err := r.Remote("origin")
	if err != nil {
		return false, fmt.Errorf("failed to get GIT remove origin: %w", err)
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
		return false, fmt.Errorf("failed to get references on the remote repository: %w", err)
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

	log.Info("branch existence status", logBranchNameKey, branchName, "existBranchOrNot", existBranchOrNot)

	return existBranchOrNot, nil
}
