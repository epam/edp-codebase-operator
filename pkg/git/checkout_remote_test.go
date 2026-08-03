package v2

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func seedCommit(t *testing.T, originDir, fileContent, pushRefSpec string) plumbing.Hash {
	t.Helper()

	seed := t.TempDir()
	repo, err := gogit.PlainInit(seed, false)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path.Join(seed, "f.txt"), []byte(fileContent), 0o600))

	wt, err := repo.Worktree()
	require.NoError(t, err)

	_, err = wt.Add("f.txt")
	require.NoError(t, err)

	hash, err := wt.Commit("init "+fileContent, &gogit.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)

	_, err = repo.CreateRemote(&gogitconfig.RemoteConfig{Name: "origin", URLs: []string{originDir}})
	require.NoError(t, err)
	require.NoError(t, repo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gogitconfig.RefSpec{gogitconfig.RefSpec(pushRefSpec)},
		Force:      true,
	}))

	return hash
}

func buildOriginWithClone(t *testing.T) (string, string) {
	t.Helper()

	originDir := t.TempDir()
	_, err := gogit.PlainInit(originDir, true)
	require.NoError(t, err)

	seedCommit(t, originDir, "base", "+refs/heads/master:refs/heads/master")

	workDir := path.Join(t.TempDir(), "clone")
	gp := NewGitProvider(Config{})
	require.NoError(t, gp.Clone(context.Background(), originDir, workDir))

	return originDir, workDir
}

func workdirHead(t *testing.T, workDir string) *plumbing.Reference {
	t.Helper()

	repo, err := gogit.PlainOpen(workDir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	return head
}

// A branch pushed to origin after the workdir was cloned is fetched into
// refs/heads by Checkout's refs/*:refs/* refspec; the existence check must look
// there, or the checkout collides on Create with the ref the fetch just wrote.
func TestCheckout_RemoteBranchCreatedAfterClone(t *testing.T) {
	originDir, workDir := buildOriginWithClone(t)

	origin, err := gogit.PlainOpen(originDir)
	require.NoError(t, err)

	masterRef, err := origin.Reference(plumbing.NewBranchReferenceName("master"), false)
	require.NoError(t, err)
	require.NoError(t, origin.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewBranchReferenceName("late-branch"), masterRef.Hash()),
	))

	gp := NewGitProvider(Config{})
	err = gp.Checkout(context.Background(), workDir, "late-branch", true)
	require.NoError(t, err, "checkout of a branch created after clone must succeed")

	require.Equal(t, "late-branch", workdirHead(t, workDir).Name().Short())
}

// TestCheckout_RemoteBranchPresentAtCloneTime covers the branchToCopy flow in
// its common shape: the target branch existed when the workdir was cloned.
func TestCheckout_RemoteBranchPresentAtCloneTime(t *testing.T) {
	originDir := t.TempDir()
	_, err := gogit.PlainInit(originDir, true)
	require.NoError(t, err)

	seedCommit(t, originDir, "base", "+refs/heads/master:refs/heads/master")
	seedCommit(t, originDir, "base", "+refs/heads/master:refs/heads/feature")

	workDir := path.Join(t.TempDir(), "clone")
	gp := NewGitProvider(Config{})
	require.NoError(t, gp.Clone(context.Background(), originDir, workDir))

	require.NoError(t, gp.Checkout(context.Background(), workDir, "feature", true))
	require.Equal(t, "feature", workdirHead(t, workDir).Name().Short())
}

// TestCheckout_BranchAbsentEverywhere preserves the create-fallback: a branch
// that exists neither locally nor on the remote is created from HEAD.
func TestCheckout_BranchAbsentEverywhere(t *testing.T) {
	_, workDir := buildOriginWithClone(t)

	gp := NewGitProvider(Config{})
	require.NoError(t, gp.Checkout(context.Background(), workDir, "brand-new", true))
	require.Equal(t, "brand-new", workdirHead(t, workDir).Name().Short())
}

// TestCheckout_ForcePushedRemoteBranch: a rewritten upstream branch must not
// fail the fetch of a cached workdir; the checkout lands on the new tip.
func TestCheckout_ForcePushedRemoteBranch(t *testing.T) {
	originDir, workDir := buildOriginWithClone(t)

	rewritten := seedCommit(t, originDir, "rewritten-history", "+refs/heads/master:refs/heads/master")

	gp := NewGitProvider(Config{})
	require.NoError(t, gp.Checkout(context.Background(), workDir, "master", true),
		"force-pushed upstream branch must not fail the checkout fetch")
	require.Equal(t, rewritten, workdirHead(t, workDir).Hash())
}
