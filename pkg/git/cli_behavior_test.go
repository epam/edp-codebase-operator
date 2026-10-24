package v2

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	allowedProtocols += ":file"
}

func requireBinary(t *testing.T, name string) {
	t.Helper()

	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not on PATH", name)
	}
}

// newBareRemote creates a bare repository whose main branch holds one commit and which has one
// extra branch per name in branches. The returned value is the file:// URL.
func newBareRemote(t *testing.T, branches ...string) (root, bare string) {
	t.Helper()

	root = t.TempDir()
	seed := filepath.Join(root, "seed")
	bare = filepath.Join(root, "remote.git")

	gitRun(t, "init", "-b", "main", seed)
	require.NoError(t, os.WriteFile(filepath.Join(seed, "a.txt"), []byte("a"), 0o600))
	gitRun(t, "-C", seed, "add", ".")
	gitRun(t, "-C", seed, "commit", "-m", "init")

	for _, b := range branches {
		gitRun(t, "-C", seed, "branch", b)
	}

	gitRun(t, "clone", "--bare", seed, bare)

	return root, bare
}

func TestSSHCommand_UserOverridesURLUser(t *testing.T) {
	requireBinary(t, "ssh")

	hostsDir := filepath.Join(t.TempDir(), "my certs")
	t.Setenv("SSH_KNOWN_HOSTS", filepath.Join(hostsDir, "known_hosts"))

	env, cleanup, err := NewCLIGitProvider(Config{SSHKey: testSSHKey(t), SSHUser: "gerrit"}).buildEnv("")
	require.NoError(t, err)

	defer cleanup()

	for _, destination := range []string{"git@example.com", "example.com"} {
		script := envMap(env)["GIT_SSH_COMMAND"] + " -G " + destination
		out, err := exec.Command("sh", "-c", script).CombinedOutput() //nolint:gosec // test input
		require.NoError(t, err, string(out))

		assert.Contains(t, string(out), "\nuser gerrit\n", "destination %s", destination)
		assert.Contains(t, string(out), "\nuserknownhostsfile "+filepath.Join(hostsDir, "known_hosts")+"\n")
		assert.Contains(t, string(out), "\nglobalknownhostsfile "+os.DevNull+"\n")
	}
}

func TestCLIGitProvider_Clone_EmptyRemote(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root := t.TempDir()
	bare := filepath.Join(root, "empty.git")
	gitRun(t, "init", "--bare", "-b", "main", bare)

	dst := filepath.Join(root, "work")
	err := NewCLIGitProvider(Config{}).Clone(context.Background(), "file://"+bare, dst)

	require.ErrorIs(t, err, transport.ErrEmptyRemoteRepository)
	assert.NoDirExists(t, dst)
}

func TestCLIGitProvider_Clone_RemoteHeadMissing(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root, bare := newBareRemote(t, "dev")
	gitRun(t, "-C", bare, "symbolic-ref", "HEAD", "refs/heads/gone")

	dst := filepath.Join(root, "work")
	err := NewCLIGitProvider(Config{}).Clone(context.Background(), "file://"+bare, dst)

	require.ErrorContains(t, err, "remote HEAD does not point to an existing branch")
	assert.NotErrorIs(t, err, transport.ErrEmptyRemoteRepository)
	assert.NoDirExists(t, dst)
}

func TestCLIGitProvider_Clone_KeepsOccupiedDestination(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root, bare := newBareRemote(t)
	dst := filepath.Join(root, "work")
	require.NoError(t, os.MkdirAll(dst, 0o755)) //nolint:gosec // test directory
	keep := filepath.Join(dst, "keep.txt")
	require.NoError(t, os.WriteFile(keep, []byte("keep"), 0o600))

	err := NewCLIGitProvider(Config{}).Clone(context.Background(), "file://"+bare, dst)

	require.Error(t, err)
	assert.FileExists(t, keep)
}

func TestCLIGitProvider_Clone_IntoEmptyDirectory(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root, bare := newBareRemote(t)
	dst := filepath.Join(root, "work")
	require.NoError(t, os.MkdirAll(dst, 0o755)) //nolint:gosec // test directory

	require.NoError(t, NewCLIGitProvider(Config{}).Clone(context.Background(), "file://"+bare, dst))
	assert.FileExists(t, filepath.Join(dst, "a.txt"))
}

func TestCLIGitProvider_Commit(t *testing.T) {
	requireBinary(t, "git")

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Init(ctx, dir))

	// Unborn HEAD, nothing staged: no commit.
	require.NoError(t, p.Commit(ctx, dir, "empty"))
	assert.Equal(t, "0", gitCount(t, dir))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600))
	require.NoError(t, p.Commit(ctx, dir, "first"))
	assert.Equal(t, "1", gitCount(t, dir))
	assert.Equal(t, "codebase <codebase@krci.local>", gitRun(t, "-C", dir, "log", "-1", "--format=%an <%ae>"))

	// Clean tree: no commit.
	require.NoError(t, p.Commit(ctx, dir, "clean"))
	assert.Equal(t, "1", gitCount(t, dir))

	// Deletion is staged and committed.
	require.NoError(t, os.Remove(filepath.Join(dir, "a.txt")))
	require.NoError(t, p.Commit(ctx, dir, "delete"))
	assert.Equal(t, "2", gitCount(t, dir))

	require.NoError(t, p.Commit(ctx, dir, "allow empty", CommitAllowEmpty()))
	assert.Equal(t, "3", gitCount(t, dir))
}

func gitCount(t *testing.T, dir string) string {
	t.Helper()

	cmd := exec.Command("git", "-C", dir, "rev-list", "--count", "--all")

	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")

	out, err := cmd.Output()
	require.NoError(t, err)

	return strings.TrimSpace(string(out))
}

func TestCLIGitProvider_Checkout(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	ctx := context.Background()
	root, bare := newBareRemote(t, "feature")
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Clone(ctx, "file://"+bare, dst))

	current := func() string { return gitRun(t, "-C", dst, "symbolic-ref", "--short", "HEAD") }

	// remote=true, branch present: plain switch.
	require.NoError(t, p.Checkout(ctx, dst, "feature", true))
	assert.Equal(t, "feature", current())

	// remote=true, branch absent: created from HEAD.
	require.NoError(t, p.Checkout(ctx, dst, "fresh", true))
	assert.Equal(t, "fresh", current())

	// remote=false creates the branch and fails if it exists.
	require.NoError(t, p.Checkout(ctx, dst, "local", false))
	assert.Equal(t, "local", current())
	require.Error(t, p.Checkout(ctx, dst, "local", false))
}

func TestCLIGitProvider_Checkout_BranchLookupError(t *testing.T) {
	requireBinary(t, "git")

	exists, err := NewCLIGitProvider(Config{}).branchExists(context.Background(), t.TempDir(), "main")

	require.Error(t, err, "a failing lookup is not an absent branch")
	assert.False(t, exists)
}

func TestCLIGitProvider_CheckoutRemoteBranch_NoGuess(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	ctx := context.Background()
	root, bare := newBareRemote(t, "stale")
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Clone(ctx, "file://"+bare, dst))

	// Remote deletes the branch; only the remote-tracking ref survives locally.
	gitRun(t, "-C", bare, "branch", "-D", "stale")
	gitRun(t, "-C", dst, "branch", "-D", "stale")
	require.NotEmpty(t, gitRun(t, "-C", dst, "for-each-ref", "refs/remotes/origin/stale"))

	require.Error(t, p.CheckoutRemoteBranch(ctx, dst, "stale"))
	assert.Equal(t, "main", gitRun(t, "-C", dst, "symbolic-ref", "--short", "HEAD"))

	require.Error(t, p.CreateChildBranch(ctx, dst, "stale", "child"))
}

func TestCLIGitProvider_Checkout_OptionLikeBranchName(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	ctx := context.Background()
	root, bare := newBareRemote(t)
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Clone(ctx, "file://"+bare, dst))

	for _, name := range []string{"--orphan", "--detach", "--pathspec-from-file=/etc/hosts"} {
		err := p.CheckoutRemoteBranch(ctx, dst, name)

		require.ErrorContains(t, err, "invalid reference", name)
		assert.Equal(t, "main", gitRun(t, "-C", dst, "symbolic-ref", "--short", "HEAD"))

		require.Error(t, p.CreateChildBranch(ctx, dst, name, "child"), name)
	}
}

func TestCLIGitProvider_originURL(t *testing.T) {
	requireBinary(t, "git")

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Init(ctx, dir))
	require.NoError(t, p.AddRemoteLink(ctx, dir, "ssh://git.example.com:29418/repo"))

	origin, err := p.originURL(ctx, dir)
	require.NoError(t, err)
	assert.Equal(t, "ssh://git.example.com:29418/repo", origin)

	_, err = p.originURL(ctx, filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

// stubGit puts an executable named git first on PATH.
func stubGit(t *testing.T, script string) {
	t.Helper()

	binDir := t.TempDir()
	stub := filepath.Join(binDir, "git")
	require.NoError(t, os.WriteFile(stub, []byte("#!/bin/sh\n"+script), 0o700)) //nolint:gosec // stub must be executable
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// processAlive reports whether pid runs. A zombie that PID 1 has not reaped does not count.
func processAlive(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}

	out, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()

	return err != nil || !strings.HasPrefix(strings.TrimSpace(string(out)), "Z")
}

// requireUsablePS skips the test when ps cannot report a process state.
func requireUsablePS(t *testing.T) {
	t.Helper()

	if err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(os.Getpid())).Run(); err != nil {
		t.Skipf("ps cannot report process state: %v", err)
	}
}

func readPID(t *testing.T, path string) int {
	t.Helper()

	var data []byte

	require.Eventually(t, func() bool {
		var err error

		data, err = os.ReadFile(path)

		return err == nil && strings.TrimSpace(string(data)) != ""
	}, 5*time.Second, 20*time.Millisecond)

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)

	return pid
}

func TestCLIGitProvider_exec_CancelTerminatesProcessGroup(t *testing.T) {
	requireUsablePS(t)

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	stubGit(t, "sleep 60 &\necho $! > "+pidFile+"\nwait\n")

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		_, err := NewCLIGitProvider(Config{}).exec(ctx, gitEnv(), "status")
		errCh <- err
	}()

	child := readPID(t, pidFile)
	start := time.Now()

	cancel()

	err := <-errCh

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), 10*time.Second)
	assert.Eventually(t, func() bool { return !processAlive(child) }, 15*time.Second, 20*time.Millisecond,
		"child of git must not outlive the call")
}

func TestCLIGitProvider_exec_CancelEscalatesToKill(t *testing.T) {
	requireUsablePS(t)

	oldGrace, oldDrain := terminateGrace, pipeDrainGrace
	terminateGrace, pipeDrainGrace = 300*time.Millisecond, time.Second

	t.Cleanup(func() { terminateGrace, pipeDrainGrace = oldGrace, oldDrain })

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	stubGit(t, "trap '' TERM\n(trap '' TERM; exec sleep 60) &\necho $! > "+pidFile+"\nwhile :; do sleep 1; done\n")

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		_, err := NewCLIGitProvider(Config{}).exec(ctx, gitEnv(), "status")
		errCh <- err
	}()

	child := readPID(t, pidFile)
	start := time.Now()

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(10 * time.Second):
		t.Fatal("exec did not return after cancellation")
	}

	assert.Less(t, time.Since(start), 10*time.Second)
	assert.Eventually(t, func() bool { return !processAlive(child) }, 15*time.Second, 20*time.Millisecond,
		"SIGKILL must reach children that ignore SIGTERM")
}

func TestCLIGitProvider_exec_ExitCode(t *testing.T) {
	stubGit(t, "exit 3\n")

	_, err := NewCLIGitProvider(Config{}).exec(context.Background(), gitEnv(), "status")

	require.Error(t, err)
	assert.True(t, hasExitCode(err, 3))
	assert.False(t, hasExitCode(err, 1))
	assert.False(t, errors.Is(err, context.Canceled))
}

func TestCLIGitProvider_Clone_LocalBranchesAndTags(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root, bare := newBareRemote(t, "dev", "feat/x")
	gitRun(t, "-C", bare, "tag", "v1", "main")

	dst := filepath.Join(root, "work")
	require.NoError(t, NewCLIGitProvider(Config{}).Clone(context.Background(), "file://"+bare, dst))

	assert.ElementsMatch(t, []string{"main", "dev", "feat/x"},
		strings.Fields(gitRun(t, "-C", dst, "for-each-ref", "--format=%(refname:short)", "refs/heads")))
	assert.Equal(t, "v1", gitRun(t, "-C", dst, "tag"))
	assert.Equal(t, "main", gitRun(t, "-C", dst, "symbolic-ref", "--short", "HEAD"))
	assert.Empty(t, gitRun(t, "-C", dst, "status", "--porcelain"))
}

func TestCLIGitProvider_Clone_RejectsExternalProtocols(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root := t.TempDir()
	proof := filepath.Join(root, "proof")
	dst := filepath.Join(root, "work")

	err := NewCLIGitProvider(Config{}).Clone(context.Background(), "ext::sh -c 'touch "+proof+"'", dst)

	require.Error(t, err)
	assert.NoFileExists(t, proof)
}

func TestCLIGitProvider_Push_OptionLikeRefspec(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	ctx := context.Background()
	root, bare := newBareRemote(t)
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Clone(ctx, "file://"+bare, dst))

	err := p.Push(ctx, dst, "--receive-pack=false")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--receive-pack=false", "git must read it as a refspec")
}

func TestCLIGitProvider_Push_MissingOrigin(t *testing.T) {
	requireBinary(t, "git")

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Init(ctx, dir))

	err := p.Push(ctx, dir, "refs/heads/main:refs/heads/main")
	require.ErrorContains(t, err, "failed to read origin URL")
}

func TestTailBuffer(t *testing.T) {
	b := &tailBuffer{limit: 5}

	n, err := b.Write([]byte("abc"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "abc", b.String())

	_, _ = b.Write([]byte("defgh"))
	assert.Equal(t, "defgh", b.String())

	for range 100 {
		_, _ = b.Write([]byte("xy"))
	}

	assert.Equal(t, "yxyxy", b.String())
	assert.LessOrEqual(t, len(b.data), 3*b.limit)
}

func TestCLIGitProvider_exec_LingeringChildAfterSuccess(t *testing.T) {
	oldGrace, oldDrain := terminateGrace, pipeDrainGrace
	terminateGrace, pipeDrainGrace = 100*time.Millisecond, 200*time.Millisecond

	t.Cleanup(func() { terminateGrace, pipeDrainGrace = oldGrace, oldDrain })

	requireUsablePS(t)

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	stubGit(t, "sleep 30 &\necho $! > '"+pidFile+"'\necho done\nexit 0\n")

	out, err := NewCLIGitProvider(Config{}).exec(context.Background(), gitEnv(), "status")

	require.NoError(t, err)
	assert.Equal(t, "done\n", out)

	child := readPID(t, pidFile)
	assert.Eventually(t, func() bool { return !processAlive(child) }, 15*time.Second, 20*time.Millisecond,
		"lingering child must not outlive exec")
}

func TestCLIGitProvider_exec_SignalDeathKillsChildren(t *testing.T) {
	requireUsablePS(t)

	oldGrace, oldDrain := terminateGrace, pipeDrainGrace
	terminateGrace, pipeDrainGrace = 100*time.Millisecond, 200*time.Millisecond

	t.Cleanup(func() { terminateGrace, pipeDrainGrace = oldGrace, oldDrain })

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	stubGit(t, "sleep 30 &\necho $! > '"+pidFile+"'\nkill -9 $$\n")

	_, err := NewCLIGitProvider(Config{}).exec(context.Background(), gitEnv(), "status")

	require.Error(t, err)

	child := readPID(t, pidFile)
	assert.Eventually(t, func() bool { return !processAlive(child) }, 15*time.Second, 20*time.Millisecond,
		"child must not outlive a git process that a signal ended")
}

func TestCLIGitProvider_Commit_DoesNotUseParentRepository(t *testing.T) {
	requireBinary(t, "git")

	ctx := context.Background()
	parent := filepath.Join(t.TempDir(), "parent")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Init(ctx, parent))

	child := filepath.Join(parent, "child")
	require.NoError(t, os.MkdirAll(child, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(child, "a.txt"), []byte("a"), 0o600))

	require.Error(t, p.Commit(ctx, child, "msg"))

	out, err := exec.Command("git", "-C", parent, "rev-list", "--all", "--count").Output()
	require.NoError(t, err)
	assert.Equal(t, "0", strings.TrimSpace(string(out)))
}

func TestCLIGitProvider_Commit_EmptyMessage(t *testing.T) {
	requireBinary(t, "git")

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Init(ctx, dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600))
	require.NoError(t, p.Commit(ctx, dir, ""))
	assert.Equal(t, "1", gitCount(t, dir))
}

func TestCLIGitProvider_exec_SignalDeathRemovesLocks(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o755)) //nolint:gosec // test directory

	index := filepath.Join(repo, ".git", "index.lock")
	stubGit(t, "touch '"+index+"'\nkill -9 $$\n")

	_, err := NewCLIGitProvider(Config{}).exec(context.Background(), gitEnv(), "-C", repo, "add", "--all")

	require.Error(t, err)
	assert.NoFileExists(t, index)
}
func TestCLIGitProvider_Push_DefaultsToAllBranches(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	ctx := context.Background()
	root, bare := newBareRemote(t, "dev")
	dst := filepath.Join(root, "work")
	p := NewCLIGitProvider(Config{})

	require.NoError(t, p.Clone(ctx, "file://"+bare, dst))
	require.NoError(t, p.CheckoutRemoteBranch(ctx, dst, "dev"))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "b.txt"), []byte("b"), 0o600))
	require.NoError(t, p.Commit(ctx, dst, "add b"))
	require.NoError(t, p.Push(ctx, dst))

	assert.Equal(t, gitRun(t, "-C", dst, "rev-parse", "dev"), gitRun(t, "-C", bare, "rev-parse", "dev"))
}

func TestCLIGitProvider_exec_KillRemovesLocks(t *testing.T) {
	oldGrace, oldDrain := terminateGrace, pipeDrainGrace
	terminateGrace, pipeDrainGrace = 300*time.Millisecond, time.Second

	t.Cleanup(func() { terminateGrace, pipeDrainGrace = oldGrace, oldDrain })

	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git", "refs", "heads"), 0o755)) //nolint:gosec // test directory

	index := filepath.Join(repo, ".git", "index.lock")
	ref := filepath.Join(repo, ".git", "refs", "heads", "main.lock")
	keep := filepath.Join(repo, ".git", "config")
	require.NoError(t, os.WriteFile(keep, []byte("x"), 0o600))

	stubGit(t, "trap '' TERM\ntouch '"+index+"' '"+ref+"'\nwhile :; do sleep 1; done\n")

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		_, err := NewCLIGitProvider(Config{}).exec(ctx, gitEnv(), "-C", repo, "add", "--all")
		errCh <- err
	}()

	require.Eventually(t, func() bool {
		_, err := os.Stat(ref)

		return err == nil
	}, 5*time.Second, 20*time.Millisecond)

	cancel()

	require.ErrorIs(t, <-errCh, context.Canceled)
	assert.NoFileExists(t, index)
	assert.NoFileExists(t, ref)
	assert.FileExists(t, keep)
}

func TestCLIGitProvider_Clone_FailureKeepsEmptyDestination(t *testing.T) {
	requireBinary(t, "git")

	setCADefaults(t, nil, nil)

	root := t.TempDir()
	bare := filepath.Join(root, "empty.git")
	gitRun(t, "init", "--bare", "-b", "main", bare)

	dst := filepath.Join(root, "work")
	require.NoError(t, os.MkdirAll(dst, 0o750))

	err := NewCLIGitProvider(Config{}).Clone(context.Background(), "file://"+bare, dst)

	require.ErrorIs(t, err, transport.ErrEmptyRemoteRepository)
	assert.DirExists(t, dst)

	entries, err := os.ReadDir(dst)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRepoDir(t *testing.T) {
	assert.Equal(t, "/r", repoDir([]string{"-C", "/r", "add"}))
	assert.Equal(t, "/r", repoDir([]string{"-c", "a=b", "-C", "/r", "add"}))
	assert.Empty(t, repoDir([]string{"clone", "-C", "/r"}), "-C after the subcommand is a subcommand option")
	assert.Empty(t, repoDir([]string{"status"}))
	assert.Empty(t, repoDir(nil))
}

func TestNormalizeKey(t *testing.T) {
	assert.Equal(t, "a\nb\n", normalizeKey("\r\na\r\nb\r\n\r\n"))
	assert.Equal(t, "a\nb\n", normalizeKey("a\nb"))
}
