package v2

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/sshhostkey"
)

const (
	maxCLIOutput = 4096
	// maxCapture is the most stdout and the most stderr bytes kept per git call.
	maxCapture = 1 << 20

	// OpenSSH client stderr text, matched by substring. Re-check on openssh-client upgrades.
	hostKeyFailedMsg  = "Host key verification failed"
	hostKeyChangedMsg = "REMOTE HOST IDENTIFICATION HAS CHANGED"
	hostKeyRevokedMsg = "REVOKED HOST KEY DETECTED"

	// git push stderr text for a remote without atomic push, matched by substring. Re-check on git upgrades.
	atomicUnsupportedMsg = "does not support --atomic"

	// Network stall bounds for remote operations, in seconds.
	httpStallSeconds = 600
	sshConnectSecs   = 30
	sshAliveSecs     = 15
	sshAliveMax      = 4
)

var (
	// terminateGrace is the time git and its child processes get after SIGTERM before SIGKILL.
	terminateGrace = 10 * time.Second
	// pipeDrainGrace is the time exec waits for the output pipes after the kill.
	pipeDrainGrace = 2 * time.Second
)

// Copy of certFiles and certDirectories in crypto/x509 root_linux.go. Re-sync on Go upgrades.
// Only the first readable file is used.
var (
	caFiles = []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/pki/tls/cacert.pem",
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
		"/etc/ssl/cert.pem",
	}
	caDirs = []string{"/etc/ssl/certs", "/etc/pki/tls/certs"}
)

// proxyEnv lists the proxy variables Go reads. Go prefers the uppercase name; libcurl in git
// ignores uppercase HTTP_PROXY. baseEnv passes the Go-resolved value under both names.
var proxyEnv = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}

// gitConfig is one entry of the GIT_CONFIG_COUNT/KEY_n/VALUE_n environment configuration.
type gitConfig struct{ key, value string }

// baseConfig applies to every git invocation.
//   - gc.auto=0, maintenance.auto=false: no detached repack that outlives the call.
//   - pack.threads=1: one pack thread.
//   - pack.windowMemory=100m, core.bigFileThreshold=64m: pack-objects caps the memory of its delta window,
//     and blobs over the threshold skip the delta search and are streamed.
var baseConfig = []gitConfig{
	{"gc.auto", "0"},
	{"maintenance.auto", "false"},
	{"pack.threads", "1"},
	{"pack.windowMemory", "100m"},
	{"core.bigFileThreshold", "64m"},
}

// defaultAllowedProtocols is GIT_ALLOW_PROTOCOL for remote operations.
const defaultAllowedProtocols = "http:https:ssh"

var allowedProtocols = defaultAllowedProtocols

// remoteConfig applies to git invocations that contact a remote.
// http.lowSpeedLimit/lowSpeedTime abort a transfer that moves under 1 byte/s for httpStallSeconds.
var remoteConfig = []gitConfig{
	{"http.lowSpeedLimit", "1"},
	{"http.lowSpeedTime", strconv.Itoa(httpStallSeconds)},
}

var _ Git = (*CLIGitProvider)(nil)

// CLIGitProvider runs the operations that read or write packfiles or the work tree through
// the git binary: Clone, Push, Commit, Checkout, CheckoutRemoteBranch, CreateChildBranch.
// go-git holds that data in the operator process. All other operations stay on go-git.
type CLIGitProvider struct {
	*GitProvider
}

// NewCLIGitProvider creates a CLIGitProvider with the given configuration.
func NewCLIGitProvider(cfg Config) *CLIGitProvider {
	return &CLIGitProvider{GitProvider: NewGitProvider(cfg)}
}

// Clone checks out the remote default branch and stores all remote branches as local
// branches and all tags. The work tree matches the cloned default branch.
//   - Fails with transport.ErrEmptyRemoteRepository if the remote has no branches, and
//     with an error if the remote HEAD does not resolve.
//   - A failed clone, branch creation or HEAD check removes destination, unless destination held files
//     before the call. git refuses to clone into a non-empty directory. An empty directory that existed
//     before the call is emptied and kept.
func (p *CLIGitProvider) Clone(ctx context.Context, repoURL, destination string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL, "destination", destination)
	log.Info("Cloning repository with git CLI")

	state, err := inspectDestination(destination)
	if err != nil {
		return fmt.Errorf("failed to inspect clone destination: %w", err)
	}

	if err = p.runRemote(ctx, repoURL, "clone", "--", repoURL, destination); err != nil {
		return cleanupClone(destination, state,
			fmt.Errorf("failed to clone repository: %w", remoteErr(err, repoURL)))
	}

	if err = p.requireHead(ctx, destination); err != nil {
		return cleanupClone(destination, state, fmt.Errorf("failed to clone repository: %w", err))
	}

	log.Info("Repository cloned successfully, now creating local branches")

	err = p.runLocal(ctx, "-C", destination, "fetch", "--quiet", "--update-head-ok", ".",
		"+refs/remotes/origin/*:refs/heads/*", "^refs/remotes/origin/HEAD")
	if err != nil {
		return cleanupClone(destination, state, fmt.Errorf("failed to create local branches: %w", err))
	}

	log.Info("Local branches created successfully")

	return nil
}

// requireHead fails if HEAD of the clone does not resolve to a commit.
func (p *CLIGitProvider) requireHead(ctx context.Context, directory string) error {
	err := p.runLocal(ctx, "-C", directory, "rev-parse", "--verify", "--quiet", "HEAD")
	if err == nil {
		return nil
	}

	if !hasExitCode(err, 1) {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	refs, err := p.outputLocal(ctx, "-C", directory, "for-each-ref", "--count=1",
		"--format=%(refname)", "refs/remotes/origin")
	if err != nil {
		return fmt.Errorf("failed to list remote branches: %w", err)
	}

	if strings.TrimSpace(refs) == "" {
		return transport.ErrEmptyRemoteRepository
	}

	return errors.New("remote HEAD does not point to an existing branch")
}

type destinationState int

const (
	destinationAbsent destinationState = iota
	destinationEmpty
	destinationOccupied
)

// inspectDestination reports whether path is absent, an empty directory, or a file or a
// non-empty directory.
func inspectDestination(path string) (destinationState, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return destinationAbsent, nil
	}

	if err != nil {
		return destinationAbsent, err
	}

	if !info.IsDir() {
		return destinationOccupied, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return destinationAbsent, err
	}

	if len(entries) > 0 {
		return destinationOccupied, nil
	}

	return destinationEmpty, nil
}

// cleanupClone restores destination to its state before the clone and returns cause.
// An occupied destination stays untouched; an empty directory is emptied and kept.
func cleanupClone(destination string, state destinationState, cause error) error {
	var rmErr error

	switch state {
	case destinationOccupied:
		return cause
	case destinationEmpty:
		rmErr = emptyDirectory(destination)
	case destinationAbsent:
		rmErr = os.RemoveAll(destination)
	}

	if rmErr != nil {
		return errors.Join(cause, fmt.Errorf("failed to remove %s: %w", destination, rmErr))
	}

	return cause
}

func emptyDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if err = os.RemoveAll(filepath.Join(path, e.Name())); err != nil {
			return err
		}
	}

	return nil
}

// Push pushes refspecs to origin without forcing. Without refspecs it pushes all branches.
// The push is atomic when the remote supports it: a rejected ref then leaves every ref unchanged.
// Otherwise it is a plain push, which can update some refs while the remote rejects others.
func (p *CLIGitProvider) Push(ctx context.Context, directory string, refspecs ...string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Pushing changes with git CLI")

	if len(refspecs) == 0 {
		refspecs = []string{RefSpecPushAllBranches}
	}

	origin, err := p.originURL(ctx, directory)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	push := func(options ...string) error {
		args := append(append([]string{"-C", directory, "push"}, options...), "--", "origin")

		return p.runRemote(ctx, origin, append(args, refspecs...)...)
	}

	err = push("--atomic")
	if err != nil && strings.Contains(err.Error(), atomicUnsupportedMsg) {
		log.Info("Remote does not support atomic push, pushing without it")

		err = push()
	}

	if err != nil {
		return fmt.Errorf("failed to push: %w", remoteErr(err, origin))
	}

	log.Info("Changes pushed successfully")

	return nil
}

// Commit stages all changes, including deletions, and commits them as defaultCommitName.
// Without CommitAllowEmpty, a clean work tree is not an error and creates no commit.
// Hooks and commit signing are disabled.
func (p *CLIGitProvider) Commit(ctx context.Context, directory, message string, ops ...CommitOps) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Committing changes with git CLI")

	option := &commitOps{}
	for _, applyOption := range ops {
		applyOption(option)
	}

	if err := p.runLocal(ctx, "-C", directory, "add", "--all"); err != nil {
		return fmt.Errorf("failed to add files to index: %w", err)
	}

	if !option.allowEmptyCommit {
		err := p.runLocal(ctx, "-C", directory, "diff", "--cached", "--quiet")
		if err == nil {
			log.Info("Nothing to commit, working tree clean")
			return nil
		}

		if !hasExitCode(err, 1) {
			return fmt.Errorf("failed to get status: %w", err)
		}
	}

	args := []string{
		"-C", directory,
		"-c", "core.hooksPath=/dev/null",
		"-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty-message", "--message", message,
	}
	if option.allowEmptyCommit {
		args = append(args, "--allow-empty")
	}

	if err := p.runLocal(ctx, args...); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Info("Changes committed successfully")

	return nil
}

// Checkout force-checks out branchName.
//   - remote=false: creates the branch from HEAD; fails if it exists.
//   - remote=true: fetches branches and tags from origin first; creates the branch from HEAD
//     only if it does not exist after the fetch.
func (p *CLIGitProvider) Checkout(ctx context.Context, directory, branchName string, remote bool) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName, "remote", remote)
	log.Info("Checking out branch with git CLI")

	create := true

	if remote {
		if err := p.fetchOrigin(ctx, directory); err != nil {
			return err
		}

		exists, err := p.branchExists(ctx, directory, branchName)
		if err != nil {
			return err
		}

		create = !exists
	}

	if err := p.switchBranch(ctx, directory, branchName, create); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	log.Info("Branch checked out successfully")

	return nil
}

// CheckoutRemoteBranch fetches branches and tags from origin and force-checks out branchName.
// Fails if the branch does not exist after the fetch.
func (p *CLIGitProvider) CheckoutRemoteBranch(ctx context.Context, directory, branchName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "branch", branchName)
	log.Info("Checking out remote branch with git CLI")

	if err := p.fetchOrigin(ctx, directory); err != nil {
		return err
	}

	if err := p.switchBranch(ctx, directory, branchName, false); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	log.Info("Remote branch checked out successfully")

	return nil
}

// CreateChildBranch force-checks out parentBranch and creates newBranch from it.
func (p *CLIGitProvider) CreateChildBranch(ctx context.Context, directory, parentBranch, newBranch string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory, "parent", parentBranch, "newBranch", newBranch)
	log.Info("Creating child branch with git CLI")

	if err := p.switchBranch(ctx, directory, parentBranch, false); err != nil {
		return fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	if err := p.switchBranch(ctx, directory, newBranch, true); err != nil {
		return fmt.Errorf("failed to create child branch: %w", err)
	}

	log.Info("Child branch created successfully")

	return nil
}

// fetchOrigin updates local branches and tags from origin, including the checked-out branch.
func (p *CLIGitProvider) fetchOrigin(ctx context.Context, directory string) error {
	origin, err := p.originURL(ctx, directory)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	err = p.runRemote(ctx, origin, "-C", directory, "fetch", "--quiet", "origin",
		refSpecFetchBranches, refSpecFetchTags, "--update-head-ok")
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", remoteErr(err, origin))
	}

	return nil
}

// switchBranch force-switches to branchName, or creates it from HEAD when create is set.
// Only refs/heads/<branchName> matches: no tag, no remote-tracking branch guess, and a name
// that starts with "-" is never read as an option.
func (p *CLIGitProvider) switchBranch(ctx context.Context, directory, branchName string, create bool) error {
	args := []string{"-C", directory, "switch", "--force"}
	if create {
		args = append(args, "--create", branchName)
	} else {
		args = append(args, "--no-guess", "--end-of-options", branchName)
	}

	return p.runLocal(ctx, args...)
}

func (p *CLIGitProvider) branchExists(ctx context.Context, directory, branchName string) (bool, error) {
	err := p.runLocal(ctx, "-C", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err == nil {
		return true, nil
	}

	if hasExitCode(err, 1) {
		return false, nil
	}

	return false, fmt.Errorf("failed to look up branch %s: %w", branchName, err)
}

func (p *CLIGitProvider) originURL(ctx context.Context, directory string) (string, error) {
	out, err := p.outputLocal(ctx, "-C", directory, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("failed to read origin URL: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// runRemote executes git with the provider's auth and returns an error carrying redacted output.
// remoteURL scopes the token to the remote host. Callers wrap the error with remoteErr.
func (p *CLIGitProvider) runRemote(ctx context.Context, remoteURL string, args ...string) error {
	env, cleanup, err := p.buildEnv(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}
	defer cleanup()

	_, err = p.exec(ctx, env, args...)

	return err
}

// runLocal executes git without credentials, for operations that do not contact a remote.
func (p *CLIGitProvider) runLocal(ctx context.Context, args ...string) error {
	_, err := p.exec(ctx, gitEnv(), args...)

	return err
}

func (p *CLIGitProvider) outputLocal(ctx context.Context, args ...string) (string, error) {
	return p.exec(ctx, gitEnv(), args...)
}

// exec returns stdout; on failure the error carries redacted stderr and stdout.
//
// git runs in its own process group. On context cancellation the group gets SIGTERM, so git
// removes its lock files, and SIGKILL after terminateGrace. After a SIGKILL the lock files in
// the -C repository are removed; the same applies to a git process that a signal ended.
// The error then wraps ctx.Err().
//
// When git ends with an error, or exits 0 while a child holds the output pipes, the process
// group gets SIGKILL before exec returns.
func (p *CLIGitProvider) exec(ctx context.Context, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = withCeiling(env, args)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	grace := terminateGrace
	cmd.WaitDelay = grace + pipeDrainGrace

	done := make(chan struct{})
	defer close(done)

	var killed atomic.Bool

	cmd.Cancel = func() error {
		pgid := cmd.Process.Pid

		go func() {
			select {
			case <-done:
			case <-time.After(grace):
				killed.Store(true)

				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			}
		}()

		return syscall.Kill(-pgid, syscall.SIGTERM)
	}

	stdout, stderr := &tailBuffer{limit: maxCapture}, &tailBuffer{limit: maxCapture}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	lingering := errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil && cmd.ProcessState.Success()
	if lingering {
		err = nil
	}

	if (err != nil || lingering) && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	if err != nil {
		if killed.Load() || diedBySignal(err) {
			removeLocks(args)
		}

		err = cliError(err, p.redact(stderr.String()+stdout.String()))
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = fmt.Errorf("%w: %w", ctxErr, err)
		}

		return "", err
	}

	return stdout.String(), nil
}

// hasExitCode reports whether err comes from a git process that exited with code.
func hasExitCode(err error, code int) bool {
	var exitErr *exec.ExitError

	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}

// tailBuffer keeps the last limit bytes written to it.
type tailBuffer struct {
	limit int
	data  []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if len(b.data) > 2*b.limit {
		b.data = append(b.data[:0], b.data[len(b.data)-b.limit:]...)
	}

	return len(p), nil
}

func (b *tailBuffer) String() string {
	if len(b.data) > b.limit {
		return string(b.data[len(b.data)-b.limit:])
	}

	return string(b.data)
}

// diedBySignal reports whether err comes from a git process that a signal ended.
func diedBySignal(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)

	return ok && status.Signaled()
}

// repoDir returns the directory named by a -C argument that precedes the git subcommand, or "".
func repoDir(args []string) string {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-C" && i+1 < len(args):
			return args[i+1]
		case args[i] == "-c":
			i++
		case !strings.HasPrefix(args[i], "-"):
			return ""
		}
	}

	return ""
}

// withCeiling stops git from searching parent directories for a repository when the -C
// directory is not one.
func withCeiling(env, args []string) []string {
	dir := repoDir(args)
	if dir == "" {
		return env
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return env
	}

	return append(env[:len(env):len(env)], "GIT_CEILING_DIRECTORIES="+filepath.Dir(abs))
}

// removeLocks deletes the *.lock files of the repository named by a leading -C argument:
// the files in .git and every *.lock under .git/refs. Call it only when no git process of
// that repository runs.
func removeLocks(args []string) {
	dir := repoDir(args)
	if dir == "" {
		return
	}

	gitDir := filepath.Join(dir, ".git")

	if entries, err := os.ReadDir(gitDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".lock") {
				_ = os.Remove(filepath.Join(gitDir, e.Name()))
			}
		}
	}

	_ = filepath.WalkDir(filepath.Join(gitDir, "refs"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".lock") {
			_ = os.Remove(path)
		}

		return nil
	})
}

// cliError maps ssh host key failures to knownhosts errors so remoteErr can enrich them.
func cliError(err error, output string) error {
	output = truncate(strings.TrimSpace(output), maxCLIOutput)
	base := fmt.Errorf("git: %w: %s", err, output)

	if !strings.Contains(output, hostKeyFailedMsg) {
		return base
	}

	if strings.Contains(output, hostKeyRevokedMsg) {
		return fmt.Errorf("%w: %w", &knownhosts.RevokedError{}, base)
	}

	keyErr := &knownhosts.KeyError{}
	if strings.Contains(output, hostKeyChangedMsg) {
		keyErr.Want = []knownhosts.KnownKey{{}}
	}

	return fmt.Errorf("%w: %w", keyErr, base)
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return "..." + s[len(s)-limit:]
}

func (p *CLIGitProvider) redact(s string) string {
	secrets := p.secrets()
	if len(secrets) == 0 {
		return s
	}

	pairs := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		pairs = append(pairs, secret, "[REDACTED]")
	}

	return strings.NewReplacer(pairs...).Replace(s)
}

func (p *CLIGitProvider) secrets() []string {
	var secrets []string

	if key := strings.TrimSpace(p.config.SSHKey); key != "" {
		secrets = append(secrets, key)
	}

	if p.config.Token != "" {
		secrets = append(secrets, p.config.Token, basicAuth(p.tokenUsername(), p.config.Token))
	}

	return secrets
}

func basicAuth(user, token string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + token))
}

// gitEnv returns the process environment for git with baseConfig and extra as environment config.
func gitEnv(extra ...gitConfig) []string {
	env := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"HOME=" + os.TempDir(),
		"TMPDIR=" + os.TempDir(),
		"LC_ALL=C",
		"GIT_AUTHOR_NAME=" + defaultCommitName,
		"GIT_AUTHOR_EMAIL=" + defaultCommitEmail,
		"GIT_COMMITTER_NAME=" + defaultCommitName,
		"GIT_COMMITTER_EMAIL=" + defaultCommitEmail,
	}

	if path, ok := os.LookupEnv("PATH"); ok {
		env = append(env, "PATH="+path)
	}

	for _, name := range proxyEnv {
		lower := strings.ToLower(name)

		value := os.Getenv(name)
		if value == "" {
			value = os.Getenv(lower)
		}

		if value != "" {
			env = append(env, name+"="+value, lower+"="+value)
		}
	}

	settings := append(append([]gitConfig{}, baseConfig...), extra...)
	env = append(env, fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(settings)))

	for i, s := range settings {
		env = append(env, fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", i, s.key), fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", i, s.value))
	}

	return env
}

// buildEnv returns the process environment for git. The SSH key wins over the token.
// An SSH remote gets the SSH command with or without a key.
// The token header applies only to remoteURL with scheme http or https.
// The CA bundle is written only for https remotes. cleanup removes the temporary files
// and must always be called.
func (p *CLIGitProvider) buildEnv(remoteURL string) (env []string, cleanup func(), err error) {
	var files []string

	cleanup = func() {
		for _, f := range files {
			_ = os.Remove(f)
		}
	}

	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	if p.config.SSHKey != "" || isSSHRemote(remoteURL) {
		keyFile := ""

		if p.config.SSHKey != "" {
			if _, perr := ssh.ParsePrivateKey([]byte(p.config.SSHKey)); perr != nil {
				return nil, cleanup, fmt.Errorf("failed to parse SSH private key: %w", perr)
			}

			var werr error

			keyFile, werr = writeTempFile("sshkey", normalizeKey(p.config.SSHKey), "/dev/shm", "")
			if werr != nil {
				return nil, cleanup, fmt.Errorf("failed to write SSH key: %w", werr)
			}

			files = append(files, keyFile)
		}

		env = append(gitEnv(remoteConfig...), "GIT_ALLOW_PROTOCOL="+allowedProtocols,
			"GIT_SSH_COMMAND="+p.sshCommand(keyFile))

		return env, cleanup, nil
	}

	extra := append([]gitConfig{}, remoteConfig...)

	if scope := httpScope(remoteURL); p.config.Token != "" && scope != "" {
		extra = append(extra, gitConfig{
			"http." + scope + ".extraHeader", "Authorization: Basic " + basicAuth(p.tokenUsername(), p.config.Token),
		})
	}

	env = append(gitEnv(extra...), "GIT_ALLOW_PROTOCOL="+allowedProtocols)

	if !strings.HasPrefix(remoteURL, "https://") {
		return env, cleanup, nil
	}

	bundle, berr := writeCABundle()
	if berr != nil {
		return nil, cleanup, berr
	}

	if bundle != "" {
		files = append(files, bundle)
		env = append(env, "GIT_SSL_CAINFO="+bundle)
	}

	return env, cleanup, nil
}

// sshCommand builds GIT_SSH_COMMAND. With a key file, -l overrides the user inside the URL.
// Without one, the URL user applies and ssh offers no identity.
// With SSH_KNOWN_HOSTS set, the files it lists are the only host key source.
func (p *CLIGitProvider) sshCommand(keyFile string) string {
	parts := []string{"ssh"}

	if keyFile != "" {
		parts = append(parts,
			"-i", shellQuote(keyFile),
			"-l", shellQuote(p.config.SSHUser),
			"-o", "IdentitiesOnly=yes",
		)
	}

	parts = append(parts,
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "ConnectTimeout="+strconv.Itoa(sshConnectSecs),
		"-o", "ServerAliveInterval="+strconv.Itoa(sshAliveSecs),
		"-o", "ServerAliveCountMax="+strconv.Itoa(sshAliveMax),
	)

	if hosts := filepath.SplitList(sshhostkey.KnownHostsFile()); len(hosts) > 0 {
		parts = append(parts,
			"-o", shellQuote("UserKnownHostsFile="+quoteHostsFiles(hosts)),
			"-o", shellQuote("GlobalKnownHostsFile="+os.DevNull),
		)
	}

	return strings.Join(append(parts,
		"-o", "HostKeyAlgorithms=+ssh-rsa",
		"-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
	), " ")
}

// quoteHostsFiles formats paths as ssh_config values: each path double-quoted, % escaped.
func quoteHostsFiles(paths []string) string {
	quoted := make([]string, len(paths))

	for i, path := range paths {
		path = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "%", "%%").Replace(path)
		quoted[i] = `"` + path + `"`
	}

	return strings.Join(quoted, " ")
}

// normalizeKey returns the key with LF line endings and one trailing newline.
// OpenSSH rejects CRLF, which Go's key parser accepts.
func normalizeKey(key string) string {
	return strings.TrimSpace(strings.ReplaceAll(key, "\r\n", "\n")) + "\n"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isSSHRemote reports whether rawURL uses the ssh transport, as ssh:// or scp-like syntax.
func isSSHRemote(rawURL string) bool {
	ep, err := transport.NewEndpoint(rawURL)

	return err == nil && ep.Protocol == "ssh"
}

// httpScope returns the git config subsection that matches every URL on the host of rawURL,
// or "" when rawURL does not use http or https.
func httpScope(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}

	return u.Scheme + "://" + u.Host + "/"
}

// writeTempFile writes content to a new file in the first directory of dirs that accepts it.
// "" is the default temporary directory.
func writeTempFile(pattern, content string, dirs ...string) (string, error) {
	if len(dirs) == 0 {
		dirs = []string{""}
	}

	var (
		f   *os.File
		err error
	)

	for _, dir := range dirs {
		if f, err = os.CreateTemp(dir, pattern); err == nil {
			break
		}
	}

	if err != nil {
		return "", err
	}

	_, werr := f.WriteString(content)
	cerr := f.Close()

	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(f.Name())

		return "", err
	}

	return f.Name(), nil
}

// writeCABundle writes the roots crypto/x509 trusts as one PEM file and returns its path,
// or "" when there are none.
//   - File: SSL_CERT_FILE, else the first readable entry of caFiles.
//   - Directories: SSL_CERT_DIR, else caDirs; every file, no OpenSSL hashed names required.
//   - Blocks that do not parse as X.509 certificates are dropped; duplicates are written once.
func writeCABundle() (string, error) {
	files := caFiles
	if f := os.Getenv("SSL_CERT_FILE"); f != "" {
		files = []string{f}
	}

	dirs := caDirs
	if d := os.Getenv("SSL_CERT_DIR"); d != "" {
		dirs = filepath.SplitList(d)
	}

	var (
		bundle bytes.Buffer
		seen   = make(map[string]struct{})
	)

	appendCerts := func(data []byte) {
		for {
			var block *pem.Block

			block, data = pem.Decode(data)
			if block == nil {
				return
			}

			if block.Type != "CERTIFICATE" {
				continue
			}

			if _, ok := seen[string(block.Bytes)]; ok {
				continue
			}

			seen[string(block.Bytes)] = struct{}{}

			if _, err := x509.ParseCertificate(block.Bytes); err != nil {
				continue
			}

			_ = pem.Encode(&bundle, &pem.Block{Type: "CERTIFICATE", Bytes: block.Bytes})
		}
	}

	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil {
			appendCerts(data)
			break
		}
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if data, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
				appendCerts(data)
			}
		}
	}

	if bundle.Len() == 0 {
		return "", nil
	}

	path, err := writeTempFile("cabundle", bundle.String())
	if err != nil {
		return "", fmt.Errorf("failed to write CA bundle: %w", err)
	}

	return path, nil
}
