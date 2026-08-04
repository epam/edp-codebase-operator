package gitserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_sshInitFromSecret_verifiesHostKeys(t *testing.T) {
	knownHosts := filepath.Join(t.TempDir(), "ssh_known_hosts")
	require.NoError(t, os.WriteFile(knownHosts, []byte(
		"[git.example.com]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n",
	), 0o600))
	t.Setenv("SSH_KNOWN_HOSTS", knownHosts)

	client, err := sshInitFromSecret(gitSshData{
		Host: "git.example.com",
		User: "git",
		Key:  testKey,
		Port: 2222,
	}, logr.Discard())
	require.NoError(t, err)

	// The connection must never be established without a host key check.
	require.NotNil(t, client.Config.HostKeyCallback)
	// Algorithms must be narrowed to what is on file, otherwise a server offering
	// an unpinned key type fails as a mismatch rather than as a missing entry.
	assert.Equal(t, []string{"ssh-ed25519"}, client.Config.HostKeyAlgorithms)
}

func Test_sshInitFromSecret_failsWithoutKnownHosts(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", filepath.Join(t.TempDir(), "absent"))

	client, err := sshInitFromSecret(gitSshData{
		Host: "git.example.com",
		User: "git",
		Key:  testKey,
		Port: 22,
	}, logr.Discard())

	require.Error(t, err)
	assert.Nil(t, client)
	// This message reaches the user through GitServer .status.error.
	assert.Contains(t, err.Error(), "known_hosts")
}

func Test_publicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "success",
			key:     testKey,
			wantErr: assert.NoError,
		},
		{
			name:    "success",
			key:     "invalid-key",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := publicKey(tt.key)
			tt.wantErr(t, err)
		})
	}
}
