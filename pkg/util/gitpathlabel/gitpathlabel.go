// Package gitpathlabel computes the platform-wide mapping key from a VCS
// repository path to its Codebase resource, stored in the
// api/v1.GitUrlPathHashLabel label. The codebase-operator stamps the label on
// every reconcile; consumers (e.g. the edp-tekton interceptor) select
// Codebases by it instead of listing the whole namespace. The value is only a
// selection filter: consumers must verify candidates against spec.gitUrlPath
// (case-insensitively) after selecting by the label.
package gitpathlabel

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// hashLen keeps 128 bits of the digest: collision-negligible at any realistic
// codebase count while always forming a valid label value (32 hex chars < 63).
const hashLen = 32

// Hash returns the canonical repository identity for a git URL path:
// the first 128 bits (hex) of sha256 over the normalized path.
//
// The normalization (leading slash, lowercase) mirrors the matching semantics
// the platform has always used (event_processor.ConvertRepositoryPath and
// strings.EqualFold in edp-tekton). This function is a frozen wire format —
// its output is persisted in labels cluster-wide and recomputed independently
// by other components; it must never change.
func Hash(gitUrlPath string) string {
	path := strings.ToLower(gitUrlPath)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	sum := sha256.Sum256([]byte(path))

	return hex.EncodeToString(sum[:])[:hashLen]
}
