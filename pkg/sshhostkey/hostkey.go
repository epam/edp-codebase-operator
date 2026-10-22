// Package sshhostkey centralises SSH host key verification for every outbound
// SSH connection the operator makes.
//
// Verification is mandatory. The known_hosts source is a single file shared by
// the whole process, located by SSH_KNOWN_HOSTS and re-read on every
// connection, so entries added to it apply to connections already in flight.
package sshhostkey

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// knownHostsEnvVar is go-git's own lookup variable, reused so that the go-git
// code paths, the raw golang.org/x/crypto/ssh code paths and the ssh binary
// (OpenSSHOptions) always agree on which files are authoritative.
const knownHostsEnvVar = "SSH_KNOWN_HOSTS"

// ClientConfig returns the host key callback and the host key algorithms to set
// on a golang.org/x/crypto/ssh.ClientConfig.
//
// Both values must be applied together. Restricting HostKeyAlgorithms to the
// types actually recorded for the host prevents the server from offering a key
// type that is absent from known_hosts, which the handshake would otherwise
// report as a key mismatch rather than as the missing entry it really is.
//
// Code that talks to a git remote through go-git must NOT use this function:
// go-git derives both values itself when AuthMethod leaves HostKeyCallback nil.
func ClientConfig(host string, port int32) (ssh.HostKeyCallback, []string, error) {
	db, err := gitssh.NewKnownHostsDb()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load SSH known_hosts (%s): %w", Source(), err)
	}

	return db.HostKeyCallback(), db.HostKeyAlgorithms(HostPort(host, int(port))), nil
}

func HostPort(host string, port int) string {
	if port == 0 {
		port = defaultSSHPort
	}

	return net.JoinHostPort(host, strconv.Itoa(port))
}

// Source is for error messages only; the file itself is opened by go-git or ssh.
func Source() string {
	if path := os.Getenv(knownHostsEnvVar); path != "" {
		return path
	}

	return "~/.ssh/known_hosts, /etc/ssh/ssh_known_hosts"
}

// OpenSSHOptions returns the ssh(1) options that verify host keys as go-git
// does: a host missing from known_hosts is refused, no key is ever added, and
// the files in SSH_KNOWN_HOSTS, when it is set, are the only source.
func OpenSSHOptions() []string {
	opts := []string{"-o", "StrictHostKeyChecking=yes", "-o", "UpdateHostKeys=no"}

	if files := filepath.SplitList(os.Getenv(knownHostsEnvVar)); len(files) > 0 {
		opts = append(opts,
			"-o", "UserKnownHostsFile="+strings.Join(files, " "),
			"-o", "GlobalKnownHostsFile=/dev/null",
		)
	}

	return opts
}

// Enrich turns a host key verification failure into an actionable message.
// Errors of any other kind are returned unchanged, so it is safe to apply to
// every error returned by an SSH operation.
func Enrich(err error, hostPort string) error {
	if err == nil {
		return nil
	}

	var revokedErr *knownhosts.RevokedError
	if errors.As(err, &revokedErr) {
		return revoked(err, hostPort)
	}

	var keyErr *knownhosts.KeyError
	if !errors.As(err, &keyErr) {
		return err
	}

	// A KeyError carrying known keys means the host is on file but presented a
	// different key: either the server was rekeyed, or the connection is being
	// intercepted. Never suggest "just add the key" for this case.
	if len(keyErr.Want) > 0 {
		return mismatch(err, hostPort)
	}

	return missing(err, hostPort)
}

// EnrichOutput is Enrich for the OpenSSH client, which reports a failed host
// key check only as text; output is what ssh printed.
func EnrichOutput(err error, output, hostPort string) error {
	switch {
	case err == nil || !strings.Contains(output, "Host key verification failed"):
		return err
	case strings.Contains(output, "REVOKED HOST KEY"):
		return revoked(err, hostPort)
	case strings.Contains(output, "REMOTE HOST IDENTIFICATION HAS CHANGED"):
		return mismatch(err, hostPort)
	default:
		return missing(err, hostPort)
	}
}

func revoked(err error, hostPort string) error {
	return fmt.Errorf(
		"SSH host key for %s is marked as revoked in known_hosts (%s), refusing to connect: %w",
		hostPort, Source(), err,
	)
}

func mismatch(err error, hostPort string) error {
	return fmt.Errorf(
		"SSH host key mismatch for %s: the server presented a key that does not match known_hosts (%s). "+
			"This means either the git server's host key was rotated, or the connection is being intercepted. "+
			"Verify the new key out-of-band before replacing the entry: %w",
		hostPort, Source(), err,
	)
}

func missing(err error, hostPort string) error {
	return fmt.Errorf(
		"SSH host key for %s is not present in known_hosts (%s). "+
			"Add it to the operator's ssh-known-hosts ConfigMap, for example: ssh-keyscan -p %s %s: %w",
		hostPort, Source(), portOf(hostPort), hostOf(hostPort), err,
	)
}

func hostOf(hostPort string) string {
	if host, _, err := net.SplitHostPort(hostPort); err == nil {
		return host
	}

	return hostPort
}

func portOf(hostPort string) string {
	if _, port, err := net.SplitHostPort(hostPort); err == nil {
		return port
	}

	return strconv.Itoa(defaultSSHPort)
}

const defaultSSHPort = 22

// IsVerificationError reports whether err was caused by host key verification
// rather than by connectivity, authentication or the git protocol.
func IsVerificationError(err error) bool {
	var keyErr *knownhosts.KeyError

	var revokedErr *knownhosts.RevokedError

	if errors.As(err, &keyErr) || errors.As(err, &revokedErr) {
		return true
	}

	// go-git reports a missing or unreadable known_hosts file as a plain error.
	return err != nil && strings.Contains(err.Error(), "known_hosts")
}
