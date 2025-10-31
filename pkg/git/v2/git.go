package v2

import "context"

// Git interface provides methods for working with git using v2 GitProvider.
// This interface uses context-aware methods and handles authentication via Config.
type Git interface {
	// Clone clones a repository to the specified destination with full history.
	Clone(ctx context.Context, repoURL, destination string) error

	// Commit commits changes in the working directory.
	Commit(ctx context.Context, directory, message string, ops ...CommitOps) error

	// Push pushes changes to the remote repository.
	// refspecs: optional refspecs (e.g., RefSpecPushAllBranches, RefSpecPushAllTags).
	Push(ctx context.Context, directory string, refspecs ...string) error

	// Checkout checks out a branch in the repository.
	// If remote is true, fetches from remote first and only creates local branch if it doesn't exist remotely.
	Checkout(ctx context.Context, directory, branchName string, remote bool) error

	// CreateRemoteBranch creates a new branch from a reference and pushes it to remote.
	// fromRef: branch name or commit hash to create from (empty string means HEAD).
	CreateRemoteBranch(ctx context.Context, directory, branchName, fromRef string) error

	// GetCurrentBranchName returns the name of the current branch.
	GetCurrentBranchName(ctx context.Context, directory string) (string, error)

	// CheckPermissions checks if the repository is accessible with current credentials.
	CheckPermissions(ctx context.Context, repoURL string) error

	// CheckReference checks if a reference (branch or commit) exists in the repository.
	CheckReference(ctx context.Context, directory, refName string) error

	// RemoveBranch removes a local branch.
	RemoveBranch(ctx context.Context, directory, branchName string) error

	// CreateChildBranch creates a new branch from an existing branch.
	CreateChildBranch(ctx context.Context, directory, parentBranch, newBranch string) error

	// Init initializes a new git repository.
	Init(ctx context.Context, directory string) error

	// Fetch fetches changes from the remote repository.
	// branchName: specific branch to fetch (empty string fetches all).
	Fetch(ctx context.Context, directory, branchName string) error

	// AddRemoteLink adds or updates the remote origin URL.
	AddRemoteLink(ctx context.Context, directory, remoteURL string) error

	// CommitExists checks if a commit with the given hash exists in the repository.
	CommitExists(ctx context.Context, directory, hash string) (bool, error)

	// CheckoutRemoteBranch fetches from remote and checks out the specified branch.
	CheckoutRemoteBranch(ctx context.Context, directory, branchName string) error

	// CreateRemoteTag creates a tag from a branch and pushes it to the remote repository.
	CreateRemoteTag(ctx context.Context, directory, branchName, tagName string) error
}
