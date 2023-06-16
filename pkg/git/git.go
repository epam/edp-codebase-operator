package git

import (
	"context"
	"errors"
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
	ctrl "sigs.k8s.io/controller-runtime"
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
	errRemoveSHHKeyFile = "failed to remove key file"
)

type commitOps struct {
	allowEmptyCommit bool
}

type CommitOps func(*commitOps)

func CommitAllowEmpty() func(*commitOps) {
	return func(o *commitOps) {
		o.allowEmptyCommit = true
	}
}

// Git interface provides methods for working with git.
//
//go:generate mockery --name Git --filename git_mock.go
type Git interface {
	CommitChanges(directory, commitMsg string, opts ...CommitOps) error
	PushChanges(key, user, directory string, port int32, pushParams ...string) error
	CheckPermissions(ctx context.Context, repo string, user, pass *string) (accessible bool)
	CloneRepositoryBySsh(ctx context.Context, key, user, repoUrl, destination string, port int32) error
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
	CommitExists(directory, hash string) (bool, error)
	AddRemoteLink(repoPath, remoteUrl string) error
}

type Command interface {
	CombinedOutput() ([]byte, error)
}

type GitProvider struct {
	CommandBuilder func(cmd string, params ...string) Command
}

func (gp *GitProvider) buildCommand(cmd string, params ...string) Command {
	if gp.CommandBuilder == nil {
		gp.CommandBuilder = func(cmd string, params ...string) Command {
			return exec.Command(cmd, params...)
		}
	}

	return gp.CommandBuilder(cmd, params...)
}

var log = ctrl.Log.WithName("git-provider")

func (gp *GitProvider) CreateRemoteBranch(key, user, p, name, fromcommit string, port int32) error {
	log.Info("start creating remote branch", logBranchNameKey, name)

	r, err := git.PlainOpen(p)
	if err != nil {
		return fmt.Errorf(errPlainOpenTmpl, p, err)
	}

	ref, err := r.Head()
	if err != nil {
		return fmt.Errorf("failed to get git HEAD reference: %w", err)
	}

	if fromcommit != "" {
		_, err = r.CommitObject(plumbing.NewHash(fromcommit))
		if err != nil {
			return fmt.Errorf("failed to get commit %s: %w", fromcommit, err)
		}

		ref = plumbing.NewReferenceFromStrings(name, fromcommit)
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

func (*GitProvider) CommitChanges(directory, commitMsg string, ops ...CommitOps) error {
	logger := log.WithValues(logDirectoryKey, directory)
	logger.Info("Start committing changes")

	option := &commitOps{
		allowEmptyCommit: false,
	}

	for _, applyOption := range ops {
		applyOption(option)
	}

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

	if !option.allowEmptyCommit {
		var status git.Status

		status, err = w.Status()
		if err != nil {
			return fmt.Errorf("failed to get git status: %w", err)
		}

		if status.IsClean() {
			logger.Info("Nothing to commit. Skip committing")

			return nil
		}
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

	logger.Info("Changes have been committed")

	return nil
}

func (gp *GitProvider) RemoveBranch(directory, branchName string) error {
	gitDir := path.Join(directory, gitDirName)
	cmd := gp.buildCommand(gitCMD, gitDirArg, gitDir, gitBranchArg, "-D", branchName)

	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove branch, err: %s: %w", string(bts), err)
	}

	return nil
}

func (gp *GitProvider) RenameBranch(directory, currentName, newName string) error {
	gitDir := path.Join(directory, gitDirName)
	cmd := gp.buildCommand(gitCMD, gitDirArg, gitDir, gitCheckoutArg, currentName)

	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch, err: %s: %w", string(bts), err)
	}

	cmd = gp.buildCommand(gitCMD, gitDirArg, gitDir, gitBranchArg, "-m", newName)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename branch, err: %s: %w", string(bts), err)
	}

	return nil
}

func (gp *GitProvider) CreateChildBranch(directory, currentBranch, newBranch string) error {
	gitDir := path.Join(directory, gitDirName)
	cmd := gp.buildCommand(gitCMD, gitDirArg, gitDir, gitCheckoutArg, currentBranch)

	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch, err: %s: %w", string(bts), err)
	}

	cmd = gp.buildCommand(gitCMD, gitDirArg, gitDir, gitCheckoutArg, "-b", newBranch)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename branch, err: %s: %w", string(bts), err)
	}

	return nil
}

func (*GitProvider) PushChanges(key, user, directory string, port int32, pushParams ...string) error {
	log.Info("Start pushing changes", logDirectoryKey, directory)

	keyPath, err := InitAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if err = os.Remove(keyPath); err != nil {
			log.Error(err, errRemoveSHHKeyFile)
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
		return fmt.Errorf("failed to push changes, err: %s: %w", string(bts), err)
	}

	log.Info("Changes has been pushed", logDirectoryKey, directory)

	return nil
}

func (*GitProvider) CheckPermissions(ctx context.Context, repo string, user, pass *string) (accessible bool) {
	l := ctrl.LoggerFrom(ctx).WithValues(logRepositoryKey, repo)

	l.Info("Checking permissions", "user", user)

	if user == nil || pass == nil {
		l.Info("No credentials provided. Skip checking permissions")
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
		},
	})
	if err != nil {
		l.Error(err, fmt.Sprintf("User %v do not have access to %v repository", user, repo))
		return false
	}

	if len(rfs) == 0 {
		l.Error(errors.New("there are not refs in repository"), "no refs in repository")
		return false
	}

	return true
}

func (*GitProvider) BareToNormal(p string) error {
	const readWriteExecutePermBits = 0o777

	gitDir := path.Join(p, gitDirName)

	if err := os.MkdirAll(gitDir, readWriteExecutePermBits); err != nil {
		return fmt.Errorf("failed to create .git folder: %w", err)
	}

	files, err := os.ReadDir(p)
	if err != nil {
		return fmt.Errorf("failed to list dir: %w", err)
	}

	for _, f := range files {
		if f.Name() == gitDirName {
			continue
		}

		oldPath := path.Join(p, f.Name())
		newPath := path.Join(p, gitDirName, f.Name())

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
	}

	cmd := exec.Command(gitCMD, gitDirArg, gitDir, "config", "--local",
		"--bool", "core.bare", "false")
	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", string(bts), err)
	}

	cmd = exec.Command(gitCMD, gitDirArg, gitDir, "config", "--local",
		"--bool", "remote.origin.mirror", "false")
	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", string(bts), err)
	}

	cmd = exec.Command(gitCMD, gitDirArg, gitDir, "reset", "--hard")
	cmd.Dir = p

	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", string(bts), err)
	}

	return nil
}

func (gp *GitProvider) CloneRepositoryBySsh(ctx context.Context, key, user, repoUrl, destination string, port int32) error {
	l := ctrl.LoggerFrom(ctx)

	l.Info("Start cloning", logRepositoryKey, repoUrl)

	keyPath, err := InitAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if err = os.Remove(keyPath); err != nil {
			l.Error(err, errRemoveSHHKeyFile)
		}
	}()

	cloneCMD := exec.Command(gitCMD, "clone", "--mirror", "--depth", "1", repoUrl, destination)
	cloneCMD.Env = []string{fmt.Sprintf(`GIT_SSH_COMMAND=ssh -i %s -l %s -o StrictHostKeyChecking=no -p %d`,
		keyPath, user, port), gitSshVariantEnv}

	bytes, err := cloneCMD.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repo by ssh, err: %s: %w", string(bytes), err)
	}

	fetchCMD := exec.Command(gitCMD, gitDirArg, destination, getFetchArg, gitUnshallowArg)
	fetchCMD.Env = cloneCMD.Env

	bts, err := fetchCMD.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch unshallow repo: %s: %w", string(bts), err)
	}

	l.Info("Result of `git fetch unshallow` command", logOutKey, string(bts))

	err = gp.BareToNormal(destination)
	if err != nil {
		return fmt.Errorf("failed to covert bare repo to normal: %w", err)
	}

	l.Info("End cloning", logRepositoryKey, repoUrl)

	return nil
}

func (gp *GitProvider) CloneRepository(repo string, user, pass *string, destination string) error {
	log.Info("Start cloning", logRepositoryKey, repo)

	const httpClientErrors = 400

	if user != nil && pass != nil {
		u, err := url.Parse(repo)
		if err != nil {
			return fmt.Errorf("failed to parse repo url: %w", err)
		}

		u.User = url.UserPassword(*user, *pass)
		repo = u.String()
	} else {
		rsp, err := netHttp.Get(repo)
		if err != nil {
			return fmt.Errorf("failed to get repo: %w", err)
		}
		if rsp.StatusCode >= httpClientErrors {
			return fmt.Errorf("repo access denied, response code: %d: %w", rsp.StatusCode, err)
		}
	}

	cloneCMD := exec.Command(gitCMD, "clone", "--mirror", "--depth", "1", repo, destination)

	if bts, err := cloneCMD.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repo: %s: %w", string(bts), err)
	}

	fetchCMD := exec.Command(gitCMD, gitDirArg, destination, getFetchArg, gitUnshallowArg)

	bts, err := fetchCMD.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch unshallow repo: %s: %w", string(bts), err)
	}

	log.Info("Result of `git fetch unshallow` command", logOutKey, string(bts))

	err = gp.BareToNormal(destination)
	if err != nil {
		return fmt.Errorf("failed to covert bare repo to normal: %w", err)
	}

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

func (*GitProvider) Fetch(key, user, workDir, branchName string) error {
	log.Info("start fetching data", logBranchNameKey, branchName)

	keyPath, err := InitAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if err = os.Remove(keyPath); err != nil {
			log.Error(err, errRemoveSHHKeyFile)
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
		return fmt.Errorf("failed to push changes, err: %s: %w", string(bts), err)
	}

	log.Info("end fetching data", logBranchNameKey, branchName)

	return nil
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
				return fmt.Errorf("failed to fetch: %w", err)
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

func (*GitProvider) CheckoutRemoteBranchBySSH(key, user, gitPath, remoteBranchName string) error {
	log.Info("start checkout to", "branch", remoteBranchName)

	keyPath, err := InitAuth(key, user)
	if err != nil {
		return err
	}

	defer func() {
		if err = os.Remove(keyPath); err != nil {
			log.Error(err, errRemoveSHHKeyFile)
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
		return fmt.Errorf("failed to fetch branches, err: %s: %w", string(bts), err)
	}

	// here we expect that remote branch exists otherwise return error
	// git checkout -b remoteBranchName remoteBranchName
	cmdCheckout := exec.Command(gitCMD, gitDirArg, gitDir, gitCheckoutArg, remoteBranchName)
	cmdCheckout.Dir = gitPath

	if bts, err := cmdCheckout.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout to branch, err: %s: %w", string(bts), err)
	}

	log.Info("end checkout to", "branch", remoteBranchName)

	return nil
}

// CommitExists checks if a commit exists in the repository.
func (*GitProvider) CommitExists(directory, hash string) (bool, error) {
	r, err := git.PlainOpen(directory)
	if err != nil {
		return false, fmt.Errorf(errPlainOpenTmpl, directory, err)
	}

	commit, err := r.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get commit object: %w", err)
	}

	return commit != nil, nil
}

// AddRemoteLink adds a remote link to the repository.
func (*GitProvider) AddRemoteLink(repoPath, remoteUrl string) error {
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

func InitAuth(key, user string) (string, error) {
	log.Info("Initializing auth", "user", user)

	keyFile, err := os.CreateTemp("", "sshkey")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for ssh key: %w", err)
	}

	keyFileInfo, _ := keyFile.Stat()
	keyFilePath := path.Join(os.TempDir(), keyFileInfo.Name())

	// write the key to the file with a new line at the end to avoid ssh errors on git commands
	// if the new line already exists, adding the new line will not cause any issues
	if _, err = fmt.Fprintf(keyFile, "%s%s", key, "\n"); err != nil {
		return "", fmt.Errorf("failed to write ssh key: %w", err)
	}

	if err = keyFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close file: %w", err)
	}

	const readOnlyPermBits = 0o400

	if err := os.Chmod(keyFilePath, readOnlyPermBits); err != nil {
		return "", fmt.Errorf("failed to chmod ssh key file: %w", err)
	}

	return keyFilePath, nil
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
		glo = &git.ListOptions{
			Auth: &http.BasicAuth{
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
