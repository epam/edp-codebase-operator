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
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	ssh2 "golang.org/x/crypto/ssh"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/sshhostkey"
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

	// Fetch refspecs: every remote branch and tag becomes a local ref. The + lets a force-pushed
	// branch or a moved tag overwrite its local copy. Do not widen to refs/*: it also downloads
	// every pull and merge request ref into memory.
	refSpecFetchAllBranches = "+refs/heads/*:refs/heads/*"
	refSpecFetchAllTags     = "+refs/tags/*:refs/tags/*"
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

		// When AuthMethod carries no HostKeyCallback, go-git loads known_hosts itself
		// and derives HostKeyAlgorithms from it. Setting a callback here suppresses
		// that derivation, so a server whose key type is absent from known_hosts
		// fails as a key mismatch instead of a missing entry.
		// See go-git plumbing/transport/ssh.command.connect.
		auth := &ssh.PublicKeys{
			User:   p.config.SSHUser,
			Signer: signer,
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

// remoteErr passes non-host-key errors through untouched, so it is safe to wrap
// every error returned by a remote operation.
func remoteErr(err error, repoURL string) error {
	return sshhostkey.Enrich(err, sshTarget(repoURL))
}

func sshTarget(repoURL string) string {
	ep, err := transport.NewEndpoint(repoURL)
	if err != nil {
		return repoURL
	}

	return sshhostkey.HostPort(ep.Host, ep.Port)
}

// originURL lets directory-based operations name the host they failed to verify.
func originURL(repo *git.Repository) string {
	remote, err := repo.Remote("origin")
	if err != nil || len(remote.Config().URLs) == 0 {
		return ""
	}

	return remote.Config().URLs[0]
}

// getTokenAuth formats token authentication based on the git provider type.
func (p *GitProvider) getTokenAuth() *http.BasicAuth {
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
	auth, err := p.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	var repo *git.Repository

	clone := func() error {
		repo, err = git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
			URL:        repoURL,
			Progress:   os.Stdout,
			Auth:       auth,
			NoCheckout: true,
		})

		return remoteErr(err, repoURL)
	}

	fetchAndCheckout := func() error {
		err := repo.FetchContext(ctx, fetchAllOptions(auth))
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to fetch all branches and tags: %w", remoteErr(err, repoURL))
		}

		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		if err := worktree.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("failed to check out the default branch: %w", err)
		}

		return nil
	}

	return cloneThenFetch(ctx, repoURL, destination, clone, fetchAndCheckout)
}

// cloneThenFetch is Clone for both engines: clone runs without a checkout, then
// fetchAndCheckout makes every remote branch and tag a local ref and checks out
// HEAD, which the fetch can move. A failure in either step removes the destination:
// callers skip a destination that is not empty and would take a partial clone, such
// as the .git a killed git leaves behind, for a complete one.
func cloneThenFetch(ctx context.Context, repoURL, destination string, clone, fetchAndCheckout func() error) error {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL, "destination", destination)
	log.Info("Cloning repository")

	if err := clone(); err != nil {
		_ = os.RemoveAll(destination)

		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info("Repository cloned successfully, now fetching all branches and tags")

	if err := fetchAndCheckout(); err != nil {
		_ = os.RemoveAll(destination)

		return err
	}

	log.Info("All branches and tags fetched successfully")

	return nil
}

// fetchAllOptions returns the options that make every branch and tag of origin a local ref.
func fetchAllOptions(auth transport.AuthMethod) *git.FetchOptions {
	return &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refSpecFetchAllBranches, refSpecFetchAllTags},
		Auth:       auth,
		Progress:   os.Stdout,
	}
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
		return fmt.Errorf("failed to push: %w", remoteErr(err, originURL(repo)))
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
		auth, err := p.getAuth()
		if err != nil {
			return fmt.Errorf("failed to get authentication: %w", err)
		}

		err = repo.FetchContext(ctx, fetchAllOptions(auth))
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to fetch: %w", remoteErr(err, originURL(repo)))
		}

		// The refspec above maps remote branches straight into local
		// refs/heads, so that is the namespace that proves existence here.
		// Checking refs/remotes/origin instead would miss any branch the
		// fetch just materialized and collide on Create below.
		_, err = repo.Reference(plumbing.NewBranchReferenceName(branchName), false)
		if err == nil {
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

// ListRemoteBranches lists branch names that exist in the remote repository
// without cloning it (equivalent to git ls-remote --heads).
func (p *GitProvider) ListRemoteBranches(ctx context.Context, repoURL string) ([]string, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL)
	log.Info("Listing remote branches")

	auth, err := p.getAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication: %w", err)
	}

	repo, _ := git.Init(memory.NewStorage(), nil)

	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create remote: %w", err)
	}

	refs, err := remote.ListContext(ctx, &git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		if errors.Is(err, transport.ErrEmptyRemoteRepository) {
			log.Info("Repository is empty, no branches found")
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list remote references: %w", remoteErr(err, repoURL))
	}

	branches := make([]string, 0, len(refs))

	for _, ref := range refs {
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
	}

	log.Info("Remote branches listed successfully", "count", len(branches))

	return branches, nil
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

		// A host key failure is neither a permission nor a lookup problem, so it
		// must not be reported as one.
		if sshhostkey.IsVerificationError(err) {
			return remoteErr(err, repoURL)
		}

		return fmt.Errorf("permission denied or repository not found: %w", err)
	}

	log.Info("Repository is accessible")

	return nil
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

	err = repo.FetchContext(ctx, fetchAllOptions(auth))
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch: %w", remoteErr(err, originURL(repo)))
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
