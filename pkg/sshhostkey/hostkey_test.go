package sshhostkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const testHost = "git.example.com"

const testPort = 2222

func newHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	key, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)

	return key
}

func writeKnownHosts(t *testing.T, key ssh.PublicKey) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "ssh_known_hosts")
	line := knownhosts.Line([]string{HostPort(testHost, testPort)}, key)

	require.NoError(t, os.WriteFile(path, []byte(line+"\n"), 0o600))
	t.Setenv(knownHostsEnvVar, path)
}

func TestHostPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{name: "explicit port", host: testHost, port: 2222, want: "git.example.com:2222"},
		{name: "zero port falls back to 22", host: testHost, port: 0, want: "git.example.com:22"},
		{name: "ipv6 is bracketed", host: "::1", port: 22, want: "[::1]:22"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, HostPort(tt.host, tt.port))
		})
	}
}

func TestClientConfig_AcceptsPinnedKey(t *testing.T) {
	key := newHostKey(t)
	writeKnownHosts(t, key)

	callback, algorithms, err := ClientConfig(testHost, testPort)
	require.NoError(t, err)
	require.NotNil(t, callback)

	// Restricting the advertised algorithms to those on file is what keeps a
	// missing entry from surfacing as a key mismatch during the handshake.
	assert.Contains(t, algorithms, key.Type())

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: testPort}
	assert.NoError(t, callback(HostPort(testHost, testPort), addr, key))
}

func TestClientConfig_RejectsUnpinnedKey(t *testing.T) {
	writeKnownHosts(t, newHostKey(t))

	callback, _, err := ClientConfig(testHost, testPort)
	require.NoError(t, err)

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: testPort}

	// A different key for a pinned host is exactly the interception case.
	err = callback(HostPort(testHost, testPort), addr, newHostKey(t))
	require.Error(t, err)

	var keyErr *knownhosts.KeyError

	require.ErrorAs(t, err, &keyErr)
	assert.NotEmpty(t, keyErr.Want, "a pinned host presenting a new key must report the expected keys")
}

func TestClientConfig_RejectsUnknownHost(t *testing.T) {
	writeKnownHosts(t, newHostKey(t))

	callback, _, err := ClientConfig(testHost, testPort)
	require.NoError(t, err)

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}

	err = callback(HostPort("other.example.com", 22), addr, newHostKey(t))
	require.Error(t, err)

	var keyErr *knownhosts.KeyError

	require.ErrorAs(t, err, &keyErr)
	assert.Empty(t, keyErr.Want, "an unpinned host has no expected keys")
}

func TestClientConfig_MissingKnownHostsFile(t *testing.T) {
	t.Setenv(knownHostsEnvVar, filepath.Join(t.TempDir(), "does-not-exist"))

	_, _, err := ClientConfig(testHost, testPort)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "known_hosts")
}

func TestEnrich(t *testing.T) {
	t.Parallel()

	otherErr := errors.New("connection refused")

	tests := []struct {
		name        string
		err         error
		wantNil     bool
		wantSame    bool
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:    "nil passes through",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "unrelated error is untouched",
			err:      otherErr,
			wantSame: true,
		},
		{
			name:        "unknown host suggests ssh-keyscan",
			err:         &knownhosts.KeyError{},
			wantContain: []string{"not present in known_hosts", "ssh-keyscan -p 2222 git.example.com"},
		},
		{
			name:        "mismatch warns about interception",
			err:         &knownhosts.KeyError{Want: []knownhosts.KnownKey{{}}},
			wantContain: []string{"mismatch", "intercepted", "out-of-band"},
			// Telling an operator to add the key would walk them into trusting
			// whatever the attacker presented.
			wantAbsent: []string{"ssh-keyscan"},
		},
		{
			name:        "revoked key is refused",
			err:         &knownhosts.RevokedError{},
			wantContain: []string{"revoked"},
			wantAbsent:  []string{"ssh-keyscan"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Enrich(tt.err, HostPort(testHost, testPort))

			if tt.wantNil {
				assert.NoError(t, got)
				return
			}

			require.Error(t, got)

			if tt.wantSame {
				assert.Equal(t, tt.err, got)
				return
			}

			for _, want := range tt.wantContain {
				assert.Contains(t, got.Error(), want)
			}

			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, got.Error(), absent)
			}

			assert.ErrorIs(t, got, tt.err, "the original error must stay in the chain")
		})
	}
}

func TestIsVerificationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "key error", err: &knownhosts.KeyError{}, want: true},
		{name: "revoked error", err: &knownhosts.RevokedError{}, want: true},
		{name: "missing known_hosts file", err: errors.New("unable to find any valid known_hosts file"), want: true},
		{name: "permission denied", err: errors.New("ssh: handshake failed"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsVerificationError(tt.err))
		})
	}
}
