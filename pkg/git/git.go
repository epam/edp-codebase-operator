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

	// GetCurrentBranchName returns the name of the current branch.
	GetCurrentBranchName(ctx context.Context, directory string) (string, error)

	// CheckPermissions checks if the repository is accessible with current credentials.
	CheckPermissions(ctx context.Context, repoURL string) error

	// ListRemoteBranches lists branch names that exist in the remote repository
	// without cloning it (equivalent to git ls-remote --heads).
	ListRemoteBranches(ctx context.Context, repoURL string) ([]string, error)

	// ResolveRemoteReference resolves a reference (branch, tag, commit hash, or empty for HEAD)
	// against the remote repository using only the reference advertisement, without cloning.
	// Returns the resolved commit hash or ErrReferenceNotFound.
	ResolveRemoteReference(ctx context.Context, repoURL, ref string) (string, error)

	// CreateRemoteBranchViaRefUpdate creates a branch on the remote pointing at fromRef
	// (branch, tag, or commit hash; must not be empty) without cloning, by sending a
	// reference update with an empty packfile. Skips creation if the branch already exists.
	CreateRemoteBranchViaRefUpdate(ctx context.Context, repoURL, branchName, fromRef string) error

	// RemoveBranch removes a local branch.
	RemoveBranch(ctx context.Context, directory, branchName string) error

	// CreateChildBranch creates a new branch from an existing branch.
	CreateChildBranch(ctx context.Context, directory, parentBranch, newBranch string) error

	// Init initializes a new git repository.
	Init(ctx context.Context, directory string) error

	// AddRemoteLink adds or updates the remote origin URL.
	AddRemoteLink(ctx context.Context, directory, remoteURL string) error

	// CheckoutRemoteBranch fetches from remote and checks out the specified branch.
	CheckoutRemoteBranch(ctx context.Context, directory, branchName string) error
}
