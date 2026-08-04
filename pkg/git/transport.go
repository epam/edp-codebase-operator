package v2

// This file implements packless git operations: reference resolution and remote
// branch creation without cloning or transferring packfiles. Memory is bounded
// by the size of the remote's reference advertisement, never by repository size,
// which makes these operations safe for arbitrarily large repositories where
// go-git clone/fetch is known to exhaust memory.
//
// NOTE: go-git v6 rewrites the plumbing/transport layer; this file must be
// ported when the dependency is upgraded.

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ErrReferenceNotFound is returned when a reference cannot be resolved in the
// remote repository's advertisement.
var ErrReferenceNotFound = errors.New("reference not found in remote repository")

// ResolveRemoteReference resolves a reference (branch name, tag name, full
// commit hash, or empty for HEAD) against the remote repository without cloning
// it, using only the reference advertisement (equivalent to git ls-remote).
// A full commit hash is returned as-is when it does not match any advertised
// reference, because reachability can only be verified by the server.
// Returns ErrReferenceNotFound if the reference cannot be resolved.
func (p *GitProvider) ResolveRemoteReference(ctx context.Context, repoURL, ref string) (string, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL, "reference", ref)
	log.Info("Resolving reference from remote advertisement")

	advRefs, closeSession, err := p.advertisedReferences(ctx, repoURL)
	if err != nil {
		return "", err
	}

	defer closeSession()

	hash, err := resolveAdvertisedRef(advRefs, ref)
	if err != nil {
		return "", err
	}

	log.Info("Reference resolved successfully", "hash", hash)

	return hash, nil
}

// CreateRemoteBranchViaRefUpdate creates a branch on the remote repository
// pointing at fromRef (branch name, tag name, or full commit hash) without
// cloning: it sends a single create command with an empty packfile, which the
// git pack protocol mandates when the server already has the target object.
// Creation is skipped if the branch already exists on the remote. fromRef must
// not be empty: the caller must pass an explicit reference (e.g. the codebase
// default branch).
//
// fromRef is resolved through the upload-pack advertisement rather than the
// receive-pack one: only upload-pack advertises peeled tag hashes, and a branch
// created from an annotated tag must point at the peeled commit — servers
// reject a branch pointing at a tag object.
func (p *GitProvider) CreateRemoteBranchViaRefUpdate(ctx context.Context, repoURL, branchName, fromRef string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("repository", repoURL, "branch", branchName, "reference", fromRef)
	log.Info("Creating remote branch via reference update")

	if fromRef == "" {
		return fmt.Errorf("fromRef must not be empty: %w", ErrReferenceNotFound)
	}

	hash, err := p.ResolveRemoteReference(ctx, repoURL, fromRef)
	if err != nil {
		return err
	}

	c, ep, auth, err := p.newTransportClient(repoURL)
	if err != nil {
		return err
	}

	session, err := c.NewReceivePackSession(ep, auth)
	if err != nil {
		return fmt.Errorf("failed to open receive-pack session: %w", remoteErr(err, repoURL))
	}

	defer func() {
		_ = session.Close()
	}()

	advRefs, err := session.AdvertisedReferencesContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get advertised references: %w", remoteErr(err, repoURL))
	}

	branchRef := plumbing.NewBranchReferenceName(branchName)
	if _, exists := advRefs.References[branchRef.String()]; exists {
		log.Info("Branch already exists on remote, skipping creation")

		return nil
	}

	req := packp.NewReferenceUpdateRequestFromCapabilities(advRefs.Capabilities)
	req.Commands = []*packp.Command{
		{Name: branchRef, Old: plumbing.ZeroHash, New: plumbing.NewHash(hash)},
	}
	req.Packfile = emptyPackfile()

	reportStatus, err := session.ReceivePack(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create remote branch %s: %w", branchName, err)
	}

	if reportStatus != nil {
		if err = reportStatus.Error(); err != nil {
			return fmt.Errorf("remote rejected branch %s creation: %w", branchName, err)
		}
	}

	log.Info("Remote branch created successfully", "hash", hash)

	return nil
}

func (p *GitProvider) newTransportClient(
	repoURL string,
) (transport.Transport, *transport.Endpoint, transport.AuthMethod, error) {
	auth, err := p.getAuth()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get authentication: %w", err)
	}

	ep, err := transport.NewEndpoint(repoURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse repository URL %q: %w", repoURL, err)
	}

	c, err := client.NewClient(ep)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create transport client: %w", err)
	}

	return c, ep, auth, nil
}

// advertisedReferences fetches the remote's reference advertisement over an
// upload-pack session that stops before any packfile negotiation; the returned
// close function must be called once the advertisement is no longer needed.
func (p *GitProvider) advertisedReferences(ctx context.Context, repoURL string) (*packp.AdvRefs, func(), error) {
	c, ep, auth, err := p.newTransportClient(repoURL)
	if err != nil {
		return nil, nil, err
	}

	session, err := c.NewUploadPackSession(ep, auth)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open upload-pack session: %w", remoteErr(err, repoURL))
	}

	advRefs, err := session.AdvertisedReferencesContext(ctx)
	if err != nil {
		_ = session.Close()

		// An empty or absent repository cannot resolve any reference; callers
		// distinguishing "not found" from transport failures rely on this.
		if errors.Is(err, transport.ErrEmptyRemoteRepository) || errors.Is(err, transport.ErrRepositoryNotFound) {
			return nil, nil, fmt.Errorf("remote repository is empty or missing: %w", ErrReferenceNotFound)
		}

		return nil, nil, fmt.Errorf("failed to get advertised references: %w", remoteErr(err, repoURL))
	}

	return advRefs, func() { _ = session.Close() }, nil
}

// resolveAdvertisedRef resolves ref within an advertisement in the order:
// HEAD (empty ref) -> branch -> tag (peeled first) -> full commit hash
// passthrough. Tag resolution prefers the peeled hash so annotated tags resolve
// to the commit they point at, matching git's behaviour.
func resolveAdvertisedRef(advRefs *packp.AdvRefs, ref string) (string, error) {
	if ref == "" {
		if advRefs.Head != nil {
			return advRefs.Head.String(), nil
		}

		return "", fmt.Errorf("remote advertisement has no HEAD: %w", ErrReferenceNotFound)
	}

	if hash, ok := advRefs.References[plumbing.NewBranchReferenceName(ref).String()]; ok {
		return hash.String(), nil
	}

	tagRef := plumbing.NewTagReferenceName(ref).String()
	if hash, ok := advRefs.Peeled[tagRef]; ok {
		return hash.String(), nil
	}

	if hash, ok := advRefs.References[tagRef]; ok {
		return hash.String(), nil
	}

	// plumbing.NewHash hex-decodes case-insensitively; comparing the canonical
	// form case-insensitively accepts uppercase hashes while rejecting inputs
	// that are not full 40-character hex strings.
	if hash := plumbing.NewHash(ref); !hash.IsZero() && strings.EqualFold(hash.String(), ref) {
		return hash.String(), nil
	}

	return "", fmt.Errorf("failed to resolve %q: %w", ref, ErrReferenceNotFound)
}

// emptyPackfile returns a packfile containing zero objects: the "PACK" header
// with version 2 and object count 0, followed by the SHA-1 checksum trailer.
// The git pack protocol requires it when a create/update command needs no
// objects because the server already has the target
// (https://git-scm.com/docs/gitprotocol-pack).
func emptyPackfile() io.ReadCloser {
	header := make([]byte, 0, 32)
	header = append(header, "PACK"...)
	header = binary.BigEndian.AppendUint32(header, 2)
	header = binary.BigEndian.AppendUint32(header, 0)

	checksum := sha1.Sum(header)

	return io.NopCloser(bytes.NewReader(append(header, checksum[:]...)))
}
