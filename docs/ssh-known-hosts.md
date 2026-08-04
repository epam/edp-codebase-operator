# SSH host key verification

Every SSH connection the codebase-operator makes to a git server is verified
against a `known_hosts` file. This includes repository clone, fetch and push,
the packless branch operations, the GitServer connectivity check, and Gerrit
administrative commands.

**Verification cannot be disabled.** An operator that accepts any host key can be
made to hand source code and credentials to whoever controls the network path.

GitServers that authenticate with a token over HTTPS are not affected — they
never open an SSH connection. Only GitServers whose credentials secret contains
an `id_rsa` key are.

## Where the host keys live

The Helm chart creates a ConfigMap named `<release>-ssh-known-hosts` and mounts
it into the operator at `/etc/codebase-operator/ssh`. The operator reads
`/etc/codebase-operator/ssh/ssh_known_hosts`, via the `SSH_KNOWN_HOSTS`
environment variable set by the chart.

The ConfigMap ships with host keys for `github.com`, `gitlab.com` and
`bitbucket.org`, so SSH GitServers pointing at those providers work with no
further configuration.

The ConfigMap is mounted as a directory rather than with `subPath`, so the
kubelet refreshes it in place and the operator re-reads it on the next
connection. **Adding a host key does not require restarting the operator.**

## Adding a self-hosted git server

Collect the server's host keys:

```sh
ssh-keyscan -t rsa,ecdsa,ed25519 git.example.com
```

For a server on a port other than 22, pass `-p` and note that the output uses
the bracket form, which is what `known_hosts` requires:

```sh
ssh-keyscan -t rsa,ecdsa,ed25519 -p 2222 git.example.com
# [git.example.com]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
```

Verify the fingerprints against a source other than the connection you just
made — your git server's documentation or its administrator:

```sh
ssh-keyscan -t rsa,ecdsa,ed25519 git.example.com | ssh-keygen -lf -
```

`ssh-keyscan` output is only as trustworthy as the network it ran over. Pinning
a key you scanned through a compromised path pins the attacker's key.

Then add the lines to the chart values:

```yaml
knownHosts:
  entries: |
    [git.example.com]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
    [git.example.com]:2222 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB...
```

Or, for an immediate fix without a Helm upgrade, edit the ConfigMap directly —
note that a subsequent `helm upgrade` will overwrite it, so fold the entry into
your values afterwards:

```sh
kubectl edit configmap <release>-ssh-known-hosts
```

## Which port to pin

Pin the port the operator actually connects on, which is not always
`spec.sshPort`:

| Operation | Port used | known_hosts entry |
|---|---|---|
| GitServer connectivity check, and all Gerrit access | `spec.sshPort` | `[git.example.com]:2222 ...` |
| Repository clone, fetch, push and branch operations for github, gitlab and bitbucket providers | always 22 | `git.example.com ...` |

The second row is a consequence of the repository URL format for those
providers: `GetSSHUrl` builds the scp-style `git@host:path.git`, which carries
no port, so the SSH client uses 22. `spec.sshPort` is honoured only for Gerrit
URLs and for the connectivity check.

**If your git server's `sshPort` is not 22, pin both forms**, or the GitServer
will report healthy while codebase operations fail (or the reverse). Scan each
port separately:

```sh
ssh-keyscan -t rsa,ecdsa,ed25519 -p 2222 git.example.com   # [git.example.com]:2222 ...
ssh-keyscan -t rsa,ecdsa,ed25519 git.example.com           # git.example.com ...
```

Both usually return the same keys, since it is normally the same server reached
on two ports.

## Diagnosing failures

A GitServer whose host key is missing or wrong reports the reason in its status:

```sh
kubectl get gitserver <name> -o jsonpath='{.status.error}'
```

| Message | Meaning | Action |
|---|---|---|
| `SSH host key for <host> is not present in known_hosts` | The server is not pinned | Add its keys as above |
| `SSH host key mismatch for <host>` | The server presented a key that differs from the pinned one | **Do not simply replace the entry.** Either the server was rekeyed or the connection is being intercepted. Confirm the new key with the server's administrator first |
| `failed to load SSH known_hosts` | The file is missing or unreadable | Check that the ConfigMap exists and is mounted |

The same messages appear on `Codebase` and `CodebaseBranch` status when the
failure happens during a repository operation.

## Upgrading from a release without host key verification

Before upgrading, list the GitServers that use SSH:

```sh
kubectl get gitservers -o json | jq -r '.items[] | select(.spec.gitProvider) |
  "\(.metadata.name) \(.spec.gitHost):\(.spec.sshPort) secret=\(.spec.nameSshKeySecret)"'
```

For each one whose secret contains an `id_rsa` key and whose host is not
github.com, gitlab.com or bitbucket.org, add its host keys to
`knownHosts.entries` as part of the upgrade. GitServers left unpinned will
report `connected: false` until their keys are added; no data is lost and the
operator recovers on the next reconcile once the entry is present.
