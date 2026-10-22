package v2

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	ssh2 "golang.org/x/crypto/ssh"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-codebase-operator/v2/pkg/sshhostkey"
)

const (
	// maxErrorOutput bounds the git output an error carries into the Codebase status.
	maxErrorOutput = 2048

	// The image's CA bundle and directory, where Go looks when SSL_CERT_FILE or SSL_CERT_DIR is unset.
	systemCertFile = "/etc/ssl/certs/ca-certificates.crt"
	systemCertDir  = "/etc/ssl/certs"
)

// credentialHelper answers git's credential requests with the token for KRCI_GIT_REMOTE, a scheme and host,
// and for nothing else.
const credentialHelper = `#!/bin/sh
[ "$1" = get ] || exit 0
while IFS== read -r key value; do
	case $key in
	protocol) protocol=$value ;;
	host) host=$value ;;
	esac
done
[ "$protocol://$host" = "$KRCI_GIT_REMOTE" ] || exit 0
printf 'username=%s\npassword=%s\n' "$KRCI_GIT_USERNAME" "$KRCI_GIT_PASSWORD"
`

// inheritedGitEnv are the only GIT_ variables git gets from the operator's environment: where it finds its
// configuration files. Any other could point git at another repository (GIT_DIR, GIT_WORK_TREE) or turn
// TLS verification off (GIT_SSL_NO_VERIFY, GIT_CONFIG_PARAMETERS).
var inheritedGitEnv = []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "GIT_CONFIG_NOSYSTEM"}

// CLIProvider runs Clone and Push with the git binary in a child process; every other method is GitProvider's.
type CLIProvider struct {
	*GitProvider
}

// NewCLIProvider creates a CLIProvider with the given configuration.
func NewCLIProvider(cfg Config) *CLIProvider {
	return &CLIProvider{GitProvider: NewGitProvider(cfg)}
}

// Clone clones a repository to the specified destination with every branch and tag.
func (p *CLIProvider) Clone(ctx context.Context, repoURL, destination string) error {
	return p.withEnv(repoURL, func(env []string) error {
		clone := func() error {
			// "--" keeps a repository URL from the Codebase spec from being read as an option.
			return run(ctx, env, "", repoURL, "clone", "--quiet", "--no-checkout", "--", repoURL, destination)
		}

		fetchAndCheckout := func() error { return fetchAllAndCheckout(ctx, env, repoURL, destination) }

		return cloneThenFetch(ctx, repoURL, destination, clone, fetchAndCheckout)
	})
}

// fetchAllAndCheckout is the second half of CLIProvider.Clone, see cloneThenFetch.
func fetchAllAndCheckout(ctx context.Context, env []string, repoURL, destination string) error {
	// git clones an empty repository, or one whose HEAD names a missing branch, without an error;
	// rev-parse --verify --quiet exits 1 for both, and go-git returns these two errors.
	if err := run(ctx, env, destination, repoURL, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			return fmt.Errorf("failed to resolve HEAD of the clone: %w", err)
		}

		if hasRefs(destination) {
			return fmt.Errorf("failed to clone repository: remote HEAD: %w", plumbing.ErrReferenceNotFound)
		}

		return fmt.Errorf("failed to clone repository: %w", transport.ErrEmptyRemoteRepository)
	}

	// --update-head-ok: the refspec also covers the branch HEAD points to.
	err := run(ctx, env, destination, repoURL,
		"fetch", "--quiet", "--update-head-ok", "origin", refSpecFetchAllBranches, refSpecFetchAllTags)
	if err != nil {
		return fmt.Errorf("failed to fetch all branches and tags: %w", err)
	}

	// go-git, which commits on this checkout later, applies no .gitattributes
	// conversion; without this file git would, and go-git would see the files as changed.
	attributes := filepath.Join(destination, ".git", "info", "attributes")
	if err := os.MkdirAll(filepath.Dir(attributes), 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", filepath.Dir(attributes), err)
	}

	if err := os.WriteFile(attributes, []byte("* -text -ident -filter -working-tree-encoding\n"), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", attributes, err)
	}

	if err := run(ctx, env, destination, repoURL, "reset", "--quiet", "--hard"); err != nil {
		return fmt.Errorf("failed to check out the default branch: %w", err)
	}

	return nil
}

// hasRefs reports whether the repository at dir holds a ref other than HEAD.
func hasRefs(dir string) bool {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return false
	}

	refs, err := repo.References()
	if err != nil {
		return false
	}

	found := false

	_ = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name() == plumbing.HEAD {
			return nil
		}

		found = true

		return storer.ErrStop
	})

	return found
}

// Push pushes the refspecs to origin; without refspecs it pushes every branch, as go-git does.
func (p *CLIProvider) Push(ctx context.Context, directory string, refspecs ...string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("directory", directory)
	log.Info("Pushing changes")

	repo, err := git.PlainOpen(directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %q: %w", directory, err)
	}

	if len(refspecs) == 0 {
		refspecs = []string{RefSpecPushAllBranches}
	}

	remoteURL := originURL(repo)

	err = p.withEnv(remoteURL, func(env []string) error {
		push := func(options ...string) error {
			args := slices.Concat([]string{"push", "--quiet"}, options, []string{"--", "origin"}, refspecs)

			return run(ctx, env, directory, remoteURL, args...)
		}

		// --atomic: every ref or none, as with go-git, which checks every update before it sends one;
		// the next reconcile would take a partial push for a complete one. A server without atomic
		// push gets a plain one.
		err := push("--atomic")
		if err != nil && strings.Contains(err.Error(), "does not support --atomic push") {
			err = push()
		}

		return err
	})
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	log.Info("Changes pushed successfully")

	return nil
}

// withEnv runs fn with the environment that passes the provider's credentials for remoteURL to git,
// never its arguments; the key and the CA bundle live in a temporary directory for the call.
func (p *CLIProvider) withEnv(remoteURL string, fn func(env []string) error) error {
	tmp, err := os.MkdirTemp("", "git-cli-")
	if err != nil {
		return fmt.Errorf("failed to create a directory for git credentials: %w", err)
	}

	defer func() { _ = os.RemoveAll(tmp) }()

	env, err := p.env(tmp, remoteURL)
	if err != nil {
		return err
	}

	return fn(env)
}

// run runs git in dir with env; remoteURL names the host in a host key error.
func run(ctx context.Context, env []string, dir, remoteURL string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = slices.Concat(inheritedEnv(), env)
	// After a cancel kills git, its ssh or git-remote-https child can still hold the output pipe.
	cmd.WaitDelay = 10 * time.Second

	out, err := cmd.CombinedOutput()
	if err != nil {
		return sshhostkey.EnrichOutput(gitError(args[0], out, err), string(out), sshTarget(remoteURL))
	}

	return nil
}

// gitError keeps the end of git's output, where git reports the failure, and
// marks rejected credentials as go-git does.
func gitError(command string, out []byte, err error) error {
	msg := string(bytes.TrimSpace(out))

	if len(msg) > maxErrorOutput {
		msg = "..." + strings.ToValidUTF8(msg[len(msg)-maxErrorOutput:], "")
	}

	if msg != "" {
		err = fmt.Errorf("git %s: %s: %w", command, msg, err)
	} else {
		err = fmt.Errorf("git %s: %w", command, err)
	}

	// With prompts off, git reports a rejected HTTP credential as a missing one.
	if strings.Contains(msg, "terminal prompts disabled") || strings.Contains(msg, "Authentication failed") {
		return fmt.Errorf("%w: %w", transport.ErrAuthenticationRequired, err)
	}

	return err
}

// inheritedEnv is the operator's environment without the GIT_ variables outside inheritedGitEnv.
func inheritedEnv() []string {
	return slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")

		return strings.HasPrefix(name, "GIT_") && !slices.Contains(inheritedGitEnv, name)
	})
}

// env passes the credentials for remoteURL to git through its environment, never its arguments.
func (p *CLIProvider) env(tmp, remoteURL string) ([]string, error) {
	env := []string{
		// Repository URLs come from the Codebase spec: allow the transports go-git
		// supports, never a remote helper such as ext::, which runs a command.
		"GIT_ALLOW_PROTOCOL=file:git:http:https:ssh",
		"GIT_TERMINAL_PROMPT=0",
	}

	sshCommand := []string{
		"ssh", "-o", "BatchMode=yes",
		"-o", "ConnectTimeout=30", "-o", "ServerAliveInterval=30", "-o", "ServerAliveCountMax=4",
	}

	if p.config.SSHKey != "" {
		keyFile := filepath.Join(tmp, "id")
		if err := writeSSHKey(keyFile, p.config.SSHKey); err != nil {
			return nil, err
		}

		// -l: the GitServer's user, which go-git uses too; Gerrit URLs carry no user.
		sshCommand = append(sshCommand, "-i", keyFile, "-o", "IdentitiesOnly=yes", "-l", p.config.SSHUser)
	}

	env = append(env, "GIT_SSH_COMMAND="+shellJoin(append(sshCommand, sshhostkey.OpenSSHOptions()...)))

	gitConfig := [][2]string{
		// The operator runs as PID 1 and does not reap the maintenance process git can detach after a fetch.
		{"maintenance.auto", "false"},
		// Bounds on git's memory: each delta thread holds a cache, and its search window holds whole
		// objects, so the threads are capped (or the memory grows with the node's CPUs), the window
		// is capped in bytes, and blobs over the threshold skip the delta search and are streamed.
		{"pack.threads", "2"},
		{"pack.windowMemory", "100m"},
		{"core.bigFileThreshold", "64m"},
		{"http.sslVerify", "true"},
	}

	// git asks the credential helper per host, so after a redirect to another host, or to plain http,
	// the helper gives it nothing, as go-git sends nothing there. The empty value drops other helpers.
	if u, err := url.Parse(remoteURL); err == nil && (u.Scheme == "https" || u.Scheme == "http") && p.config.Token != "" {
		helper := filepath.Join(tmp, "credential-helper")
		if err := os.WriteFile(helper, []byte(credentialHelper), 0o700); err != nil {
			return nil, fmt.Errorf("failed to write the git credential helper: %w", err)
		}

		auth := p.getTokenAuth()
		env = append(env,
			"KRCI_GIT_REMOTE="+u.Scheme+"://"+u.Host, "KRCI_GIT_USERNAME="+auth.Username, "KRCI_GIT_PASSWORD="+auth.Password)
		gitConfig = append(gitConfig, [2]string{"credential.helper", ""}, [2]string{"credential.helper", helper})
	}

	env = append(env, "GIT_CONFIG_COUNT="+strconv.Itoa(len(gitConfig)))
	for i, kv := range gitConfig {
		env = append(env, fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", i, kv[0]), fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", i, kv[1]))
	}

	bundle, err := caBundle(tmp)
	if err != nil {
		return nil, err
	}

	if bundle != "" {
		env = append(env, "GIT_SSL_CAINFO="+bundle)
	}

	return env, nil
}

// writeSSHKey writes the key re-encoded through x/crypto, which go-git parses keys with: ssh rejects
// spellings go-git accepts, such as CRLF line endings, surrounding blank lines or a missing final newline.
func writeSSHKey(keyFile, sshKey string) error {
	key, err := ssh2.ParseRawPrivateKey([]byte(sshKey))
	if err != nil {
		return fmt.Errorf("failed to parse SSH private key: %w", err)
	}

	// ParseRawPrivateKey returns a pointer to the ed25519 key type that MarshalPrivateKey takes as a value.
	if k, ok := key.(*ed25519.PrivateKey); ok {
		key = *k
	}

	block, err := ssh2.MarshalPrivateKey(key, "")
	if err != nil {
		return fmt.Errorf("failed to encode the SSH key for git: %w", err)
	}

	if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600); err != nil {
		return fmt.Errorf("failed to write the SSH key for git: %w", err)
	}

	return nil
}

// caBundle gives git the certificates Go trusts, as one file for GIT_SSL_CAINFO: git reads neither
// SSL_CERT_FILE nor SSL_CERT_DIR. Like Go, it takes SSL_CERT_FILE or the system bundle, plus every
// certificate in the SSL_CERT_DIR directories or the system one. It returns "" when neither variable
// is set, because git then trusts the system bundle already.
func caBundle(tmp string) (string, error) {
	certFile, certDirs := os.Getenv("SSL_CERT_FILE"), filepath.SplitList(os.Getenv("SSL_CERT_DIR"))
	if certFile == "" && len(certDirs) == 0 {
		return "", nil
	}

	if certFile == "" {
		certFile = systemCertFile
	}

	if len(certDirs) == 0 {
		certDirs = []string{systemCertDir}
	}

	files := []string{certFile}

	for _, dir := range certDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	var bundle bytes.Buffer

	seen := map[string]bool{}

	for _, file := range files {
		// Entries that are not files, such as the ..data directory of a mounted secret, fail to read.
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
			// OpenSSL rejects the whole file for one block that does not parse; Go skips the block.
			if block.Type != "CERTIFICATE" || seen[string(block.Bytes)] {
				continue
			}

			if _, err := x509.ParseCertificate(block.Bytes); err != nil {
				continue
			}

			seen[string(block.Bytes)] = true
			_ = pem.Encode(&bundle, block)
		}
	}

	if bundle.Len() == 0 {
		return "", nil
	}

	bundleFile := filepath.Join(tmp, "ca-certificates.crt")
	if err := os.WriteFile(bundleFile, bundle.Bytes(), 0o600); err != nil {
		return "", fmt.Errorf("failed to write the CA bundle for git: %w", err)
	}

	return bundleFile, nil
}

// shellJoin quotes args for GIT_SSH_COMMAND, which git runs through the shell.
func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
	}

	return strings.Join(quoted, " ")
}
