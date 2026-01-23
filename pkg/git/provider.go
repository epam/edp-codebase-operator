package v2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	ssh2 "golang.org/x/crypto/ssh"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	defaultSSHPort     = 22
	defaultSSHUser     = "git"
	defaultCommitName  = "codebase"
	defaultCommitEmail = "codebase@krci.local"

	// RefSpecPushAllBranches is the refspec for pushing all branches.
	// Equivalent to git push --all
	RefSpecPushAllBranches = "refs/heads/*:refs/heads/*"

	// RefSpecPushAllTags is the refspec for pushing all tags.
	// Equivalent to git push --tags
	RefSpecPushAllTags = "refs/tags/*:refs/tags/*"
)

// commitOps holds options for commit operations.
type commitOps struct {
	allowEmptyCommit bool
}

// CommitOps is a function that applies options to commitOps.
type CommitOps func(*commitOps)

// CommitAllowEmpty returns an option to allow empty commits.
func CommitAllowEmpty() CommitOps {
	return func(o *commitOps) {
		o.allowEmptyCommit = true
	}
}

// Config holds the configuration for GitProvider.
type Config struct {
	// GitProvider specifies the git provider type.
	// Valid values: codebaseApi.GitProviderGithub, codebaseApi.GitProviderGitlab, codebaseApi.GitProviderBitbucket
	// Used to format token authentication correctly
	GitProvider string

	// SSH authentication fields (optional)
	SSHKey  string // PEM-encoded private key
	SSHUser string // SSH username (default: "git")
	SSHPort int32  // SSH port (default: 22)

	// Token authentication fields (optional)
	Token    string // Access token for HTTP authentication
	Username string // Username for token auth (required for Bitbucket)
}

// GitProvider provides git operations using go-git library exclusively.
type GitProvider struct {
	config Config
}

// NewGitProvider creates a new GitProvider with the given configuration.
func NewGitProvider(cfg Config) *GitProvider {
	// Set defaults
	if cfg.SSHUser == "" {
		cfg.SSHUser = defaultSSHUser
	}

	if cfg.SSHPort == 0 {
		cfg.SSHPort = defaultSSHPort
	}

	return &GitProvider{
		config: cfg,
	}
}

// getAuth returns the appropriate authentication method based on configuration.
// Tries SSH first if configured, then falls back to token-based HTTP auth.
func (p *GitProvider) getAuth() (transport.AuthMethod, error) {
	// Try SSH auth first if configured
	if p.config.SSHKey != "" {
		signer, err := ssh2.ParsePrivateKey([]byte(p.config.SSHKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
		}

		auth := &ssh.PublicKeys{
			User:   p.config.SSHUser,
			Signer: signer,
			HostKeyCallbackHelper: ssh.HostKeyCallbackHelper{
				HostKeyCallback: ssh2.InsecureIgnoreHostKey(),
			},
		}

		return auth, nil
	}

	// Fall back to token-based HTTP auth if configured
	if p.config.Token != "" {
		return p.getTokenAuth(), nil
	}

	// No authentication configured
	return nil, nil
}

// getTokenAuth formats token authentication based on the git provider type.
func (p *GitProvider) getTokenAuth() transport.AuthMethod {
	switch p.config.GitProvider {
	case codebaseApi.GitProviderGithub:
		// GitHub: username=token, password=""
		return &http.BasicAuth{
			Username: p.config.Username,
			Password: p.config.Token,
		}
	case codebaseApi.GitProviderGitlab:
		// GitLab: username="oauth2", password=token
		return &http.BasicAuth{
			Username: "oauth2",
			Password: p.config.Token,
		}
	case codebaseApi.GitProviderBitbucket:
		// Bitbucket: username=username, password=app_password
		return &http.BasicAuth{
			Username: p.config.Username,
			Password: p.config.Token,
		}
	default:
		// Default to GitHub format
		return &http.BasicAuth{
			Username: p.config.Username,
			Password: p.config.Token,
		}
	}
}

// Clone clones a repository to the specified destination.
func (p *GitProvider) Clone(ctx context.Context, repoURL, destination string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL, "destination", destination)
	log.Info("Cloning repository")

	auth, err := p.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	// Clone the repository (gets default branch)
	cloneOptions := &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Auth:     auth,
	}

	repo, err := git.PlainCloneContext(ctx, destination, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info("Repository cloned successfully, now fetching all branches and tags")

	// Fetch all refs (branches, tags, etc.) to get everything similar to --mirror
	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/heads/*"),
			config.RefSpec("+refs/tags/*:refs/tags/*"),
		},
	}

	err = repo.FetchContext(ctx, fetchOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch all branches and tags: %w", err)
	}

	log.Info("All branches and tags fetched successfully")

	return nil
}

// Commit commits changes in the working directory.
func (p *GitProvider) Commit(ctx context.Context, directory, message string, ops ...CommitOps) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Committing changes")

	// Apply options
	option := &commitOps{
		allowEmptyCommit: false,
	}
	for _, applyOption := range ops {
		applyOption(option)
	}

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	err = worktree.AddWithOptions(&git.AddOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to add files to index: %w", err)
	}

	// Check if there are changes to commit
	if !option.allowEmptyCommit {
		status, err := worktree.Status()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		if status.IsClean() {
			log.Info("Nothing to commit, working tree clean")
			return nil
		}
	}

	// Commit changes
	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  defaultCommitName,
			Email: defaultCommitEmail,
			When:  time.Now(),
		},
		AllowEmptyCommits: option.allowEmptyCommit,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Info("Changes committed successfully")

	return nil
}

// Push pushes changes to the remote repository.
func (p *GitProvider) Push(ctx context.Context, directory string, refspecs ...string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Pushing changes")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	auth, err := p.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	pushOptions := &git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
	}

	// Convert refspecs if provided
	if len(refspecs) > 0 {
		pushOptions.RefSpecs = make([]config.RefSpec, len(refspecs))
		for i, refspec := range refspecs {
			pushOptions.RefSpecs[i] = config.RefSpec(refspec)
		}
	}

	err = repo.PushContext(ctx, pushOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to push: %w", err)
	}

	log.Info("Changes pushed successfully")

	return nil
}

// Checkout checks out a branch in the repository.
// If remote is true, fetches from remote first and only creates local branch if it doesn't exist remotely.
// If remote is false, checks out existing branch without fetching.
func (p *GitProvider) Checkout(ctx context.Context, directory, branchName string, remote bool) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName, "remote", remote)
	log.Info("Checking out branch")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	createBranch := true

	if remote {
		// Fetch from remote first
		auth, err := p.getAuth()
		if err != nil {
			return fmt.Errorf("failed to get authentication: %w", err)
		}

		fetchOptions := &git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*"},
			Auth:     auth,
			Progress: os.Stdout,
		}

		err = repo.FetchContext(ctx, fetchOptions)
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to fetch: %w", err)
		}

		// Check if branch exists remotely
		remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branchName)

		_, err = repo.Reference(remoteBranchRef, false)
		if err == nil {
			// Branch exists remotely, don't create locally
			createBranch = false
		}
	}

	checkoutOptions := &git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Force:  true,
		Create: createBranch,
	}

	err = worktree.Checkout(checkoutOptions)
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	log.Info("Branch checked out successfully")

	return nil
}

// CreateRemoteBranch creates a new branch from a reference and pushes it to remote.
func (p *GitProvider) CreateRemoteBranch(ctx context.Context, directory, branchName, fromRef string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName, "from", fromRef)
	log.Info("Creating remote branch")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("failed to get branches iterator: %w", err)
	}

	exists, err := branchExists(branchName, branches)
	if err != nil {
		return err
	}

	if exists {
		log.Info("Branch already exists. Skip creating")
		return nil
	}

	targetHash, err := resolveReference(repo, fromRef)
	if err != nil {
		return err
	}

	newRef := plumbing.NewHashReference(
		plumbing.NewBranchReferenceName(branchName),
		targetHash,
	)

	err = repo.Storer.SetReference(newRef)
	if err != nil {
		return fmt.Errorf("failed to set reference: %w", err)
	}

	// Push all branches
	err = p.Push(ctx, directory, RefSpecPushAllBranches)
	if err != nil {
		return err
	}

	log.Info("Remote branch created successfully")

	return nil
}

// GetCurrentBranchName returns the name of the current branch.
func (p *GitProvider) GetCurrentBranchName(ctx context.Context, directory string) (string, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Getting current branch")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return "", fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	branchName := head.Name().Short()
	log.Info("Current branch retrieved", "branch", branchName)

	return branchName, nil
}

// CheckPermissions checks if the repository is accessible with current credentials.
func (p *GitProvider) CheckPermissions(ctx context.Context, repoURL string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL)
	log.Info("Checking repository permissions")

	// If no credentials provided, assume public repository
	if p.config.SSHKey == "" && p.config.Token == "" {
		log.Info("No credentials provided, assuming public repository")
		return nil
	}

	auth, err := p.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	// Create a temporary in-memory remote to test access
	repo, _ := git.Init(memory.NewStorage(), nil)

	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	// Try to list references
	_, err = remote.ListContext(ctx, &git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		if errors.Is(err, transport.ErrEmptyRemoteRepository) {
			log.Info("Repository is empty but accessible")
			return nil
		}

		return fmt.Errorf("permission denied or repository not found: %w", err)
	}

	log.Info("Repository is accessible")

	return nil
}

// CheckReference checks if a reference (branch or commit) exists in the repository.
func (p *GitProvider) CheckReference(ctx context.Context, directory, refName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "reference", refName)
	log.Info("Checking reference")

	if refName == "" {
		return nil
	}

	r, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	_, err = resolveReference(r, refName)

	return err
}

// RemoveBranch removes a local branch.
func (p *GitProvider) RemoveBranch(ctx context.Context, directory, branchName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName)
	log.Info("Removing branch")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	err = repo.Storer.RemoveReference(plumbing.NewBranchReferenceName(branchName))
	if err != nil {
		return fmt.Errorf("failed to remove branch: %w", err)
	}

	log.Info("Branch removed successfully")

	return nil
}

// CreateChildBranch creates a new branch from an existing branch.
func (p *GitProvider) CreateChildBranch(ctx context.Context, directory, parentBranch, newBranch string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "parent", parentBranch, "newBranch", newBranch)
	log.Info("Creating child branch")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Checkout parent branch first
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(parentBranch),
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Create and checkout new branch
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(newBranch),
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create child branch: %w", err)
	}

	log.Info("Child branch created successfully")

	return nil
}

// Init initializes a new git repository.
func (p *GitProvider) Init(ctx context.Context, directory string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Initializing repository")

	_, err := git.PlainInit(directory, false)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	log.Info("Repository initialized successfully")

	return nil
}

// Fetch fetches changes from the remote repository.
func (p *GitProvider) Fetch(ctx context.Context, directory, branchName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName)
	log.Info("Fetching changes")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	auth, err := p.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
	}

	if branchName != "" {
		refSpec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName)
		fetchOptions.RefSpecs = []config.RefSpec{config.RefSpec(refSpec)}
	}

	err = repo.FetchContext(ctx, fetchOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	log.Info("Changes fetched successfully")

	return nil
}

// AddRemoteLink adds or updates the remote origin URL.
func (p *GitProvider) AddRemoteLink(ctx context.Context, directory, remoteURL string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "remoteURL", remoteURL)
	log.Info("Adding remote link")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	// Try to delete existing origin if it exists
	err = repo.DeleteRemote("origin")
	if err != nil && !errors.Is(err, git.ErrRemoteNotFound) {
		return fmt.Errorf("failed to delete existing remote: %w", err)
	}

	// Create new origin
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	log.Info("Remote link added successfully")

	return nil
}

// CommitExists checks if a commit with the given hash exists in the repository.
func (p *GitProvider) CommitExists(ctx context.Context, directory, hash string) (bool, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "hash", hash)
	log.Info("Checking if commit exists")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return false, fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	commitHash := plumbing.NewHash(hash)

	_, err = repo.CommitObject(commitHash)
	if err != nil {
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get commit: %w", err)
	}

	log.Info("Commit exists")

	return true, nil
}

// CheckoutRemoteBranch fetches from remote and checks out the specified branch.
// This is a convenience method that fetches and checks out a remote branch.
func (p *GitProvider) CheckoutRemoteBranch(ctx context.Context, directory, branchName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName)
	log.Info("Checking out remote branch")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	// Fetch from remote first
	auth, err := p.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"refs/*:refs/*"},
		Auth:       auth,
		Progress:   os.Stdout,
		Force:      true, // Equivalent to --update-head-ok
	}

	err = repo.FetchContext(ctx, fetchOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Checkout the branch (expects it exists remotely)
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	checkoutOptions := &git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Force:  true,
	}

	err = worktree.Checkout(checkoutOptions)
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	log.Info("Remote branch checked out successfully")

	return nil
}

// CreateRemoteTag creates a tag from a branch and pushes it to the remote repository.
func (p *GitProvider) CreateRemoteTag(ctx context.Context, directory, branchName, tagName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName, "tag", tagName)
	log.Info("Creating remote tag")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	// Check if tag already exists
	tags, err := repo.Tags()
	if err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}

	exists := false

	err = tags.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == tagName {
			exists = true
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate tags: %w", err)
	}

	if exists {
		log.Info("Tag already exists, skipping creation")
		return nil
	}

	// Get the branch reference
	branchRef, err := repo.Reference(plumbing.NewBranchReferenceName(branchName), false)
	if err != nil {
		return fmt.Errorf("failed to get branch reference: %w", err)
	}

	// Create the tag reference
	tagRef := plumbing.NewHashReference(plumbing.NewTagReferenceName(tagName), branchRef.Hash())

	err = repo.Storer.SetReference(tagRef)
	if err != nil {
		return fmt.Errorf("failed to create tag reference: %w", err)
	}

	// Push the tag
	err = p.Push(ctx, directory, RefSpecPushAllTags)
	if err != nil {
		return fmt.Errorf("failed to push tag: %w", err)
	}

	log.Info("Remote tag created successfully")

	return nil
}

func branchExists(branchName string, branches storer.ReferenceIter) (bool, error) {
	exist := false

	if err := branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == branchName {
			exist = true
			return storer.ErrStop
		}

		return nil
	}); err != nil {
		return false, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return exist, nil
}

// resolveReference resolves a reference (branch or commit) to a hash.
func resolveReference(r *git.Repository, ref string) (plumbing.Hash, error) {
	if ref == "" {
		// If no reference specified, use HEAD
		ref, err := r.Head()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to get git HEAD reference: %w", err)
		}

		return ref.Hash(), nil
	}

	// Try to resolve as a branch first
	branchRef, err := r.Reference(plumbing.NewBranchReferenceName(ref), false)
	if err == nil {
		return branchRef.Hash(), nil
	}

	// If not a branch, try to resolve as a commit
	commitHash := plumbing.NewHash(ref)
	if commitHash.IsZero() {
		return plumbing.ZeroHash, fmt.Errorf("invalid reference or commit hash: %s", ref)
	}

	_, err = r.CommitObject(commitHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get commit %s: %w", ref, err)
	}

	return commitHash, nil
}
