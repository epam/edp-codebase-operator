package v2

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
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

	// refSpecFetchBranches and refSpecFetchTags are the only namespaces fetched.
	// refs/pull, refs/merge-requests and refs/changes are excluded.
	refSpecFetchBranches = "+refs/heads/*:refs/heads/*"
	refSpecFetchTags     = "+refs/tags/*:refs/tags/*"
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

// GitProvider runs the go-git operations: local ref operations, init, remote link, packless remote operations.
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

// tokenUsername is the basic auth user for token auth: oauth2 for GitLab, Username otherwise.
func (p *GitProvider) tokenUsername() string {
	if p.config.GitProvider == codebaseApi.GitProviderGitlab {
		return "oauth2"
	}

	return p.config.Username
}

func (p *GitProvider) getTokenAuth() transport.AuthMethod {
	return &http.BasicAuth{
		Username: p.tokenUsername(),
		Password: p.config.Token,
	}
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
