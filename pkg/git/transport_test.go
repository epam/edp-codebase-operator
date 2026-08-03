// nolint:dupl // Duplicate test setup is acceptable in tests for readability
package v2

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testMasterHash = "6ecf0ef2c2dffb796033e5a02219af86ec6584e5"
	testBranchHash = "e8d3ffab552895c19b9fcf7aa264d277cde33881"
	testTagHash    = "b8e471f58bcbca63b07bda20e428190409c2db47"
	testPeeledHash = "9632f02833b2f9613afb5e75682132b0b22e4a31"
)

// uploadPackAdvertisement is a canned smart-HTTP ref advertisement containing
// a HEAD symref to refs/heads/master, refs/heads/branch, refs/heads/master and
// pull-request refs that resolution must ignore.
func uploadPackAdvertisement(t *testing.T) []byte {
	t.Helper()

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAxNTY2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IEhFQUQAbXVsdGlfYWNrIHRoaW4tcGFjayBzaWRlLWJhbmQgc2lkZS1iYW5kLTY0ayBvZnMtZGVsdGEgc2hhbGxvdyBkZWVwZW4tc2luY2UgZGVlcGVuLW5vdCBkZWVwZW4tcmVsYXRpdmUgbm8tcHJvZ3Jlc3MgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIGFsbG93LXRpcC1zaGExLWluLXdhbnQgYWxsb3ctcmVhY2hhYmxlLXNoYTEtaW4td2FudCBuby1kb25lIHN5bXJlZj1IRUFEOnJlZnMvaGVhZHMvbWFzdGVyIGZpbHRlciBvYmplY3QtZm9ybWF0PXNoYTEgYWdlbnQ9Z2l0L2dpdGh1Yi1nNzhiNDUyNDEzZThiCjAwM2ZlOGQzZmZhYjU1Mjg5NWMxOWI5ZmNmN2FhMjY0ZDI3N2NkZTMzODgxIHJlZnMvaGVhZHMvYnJhbmNoCjAwM2Y2ZWNmMGVmMmMyZGZmYjc5NjAzM2U1YTAyMjE5YWY4NmVjNjU4NGU1IHJlZnMvaGVhZHMvbWFzdGVyCjAwM2ViOGU0NzFmNThiY2JjYTYzYjA3YmRhMjBlNDI4MTkwNDA5YzJkYjQ3IHJlZnMvcHVsbC8xL2hlYWQKMDAzZTk2MzJmMDI4MzNiMmY5NjEzYWZiNWU3NTY4MjEzMmIwYjIyZTRhMzEgcmVmcy9wdWxsLzIvaGVhZAowMDNmYzM3ZjU4YTEzMGNhNTU1ZTQyZmY5NmEwNzFjYjljY2IzZjQzNzUwNCByZWZzL3B1bGwvMi9tZXJnZQowMDAw`) // nolint:lll
	require.NoError(t, err)

	return bts
}

// emptyUploadPackAdvertisement is the canned advertisement of an empty
// repository: capabilities only, no refs.
func emptyUploadPackAdvertisement(t *testing.T) []byte {
	t.Helper()

	bts, err := base64.StdEncoding.DecodeString(`MDAxZSMgc2VydmljZT1naXQtdXBsb2FkLXBhY2sKMDAwMDAwZGUwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIGNhcGFiaWxpdGllc157fQAgaW5jbHVkZS10YWcgbXVsdGlfYWNrX2RldGFpbGVkIG11bHRpX2FjayBvZnMtZGVsdGEgc2lkZS1iYW5kIHNpZGUtYmFuZC02NGsgdGhpbi1wYWNrIG5vLXByb2dyZXNzIHNoYWxsb3cgbm8tZG9uZSBhZ2VudD1KR2l0L3Y1LjkuMC4yMDIwMDkwODA1MDEtci00MS1nNWQ5MjVlY2JiCjAwMDA=`) // nolint:lll
	require.NoError(t, err)

	return bts
}

func staticResponseServer(t *testing.T, bts []byte) *httptest.Server {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write(bts)
		assert.NoError(t, err, "failed to write response")
	}))
	t.Cleanup(s.Close)

	return s
}

func uploadPackServer(t *testing.T) *httptest.Server {
	t.Helper()

	return staticResponseServer(t, uploadPackAdvertisement(t))
}

func emptyUploadPackServer(t *testing.T) *httptest.Server {
	t.Helper()

	return staticResponseServer(t, emptyUploadPackAdvertisement(t))
}

// buildAdvertisement encodes a smart-HTTP ref advertisement for the given
// service. Peeled entries and HEAD only appear for git-upload-pack, mirroring
// real servers: git-receive-pack advertises plain refs only.
func buildAdvertisement(t *testing.T, service string, refs, peeled map[string]string, head string) []byte {
	t.Helper()

	advRefs := packp.NewAdvRefs()
	for name, hash := range refs {
		advRefs.References[name] = plumbing.NewHash(hash)
	}

	require.NoError(t, advRefs.Capabilities.Add(capability.ReportStatus))
	require.NoError(t, advRefs.Capabilities.Add(capability.DeleteRefs))
	require.NoError(t, advRefs.Capabilities.Add(capability.OFSDelta))
	require.NoError(t, advRefs.Capabilities.Add(capability.Agent, "test/1.0"))

	if service == "git-upload-pack" {
		for name, hash := range peeled {
			advRefs.Peeled[name] = plumbing.NewHash(hash)
		}

		if head != "" {
			headHash := plumbing.NewHash(head)
			advRefs.Head = &headHash
		}
	}

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%04x# service=%s\n0000", len(service)+15+4, service)
	require.NoError(t, advRefs.Encode(buf))

	return buf.Bytes()
}

// receivePackServer serves upload-pack and receive-pack advertisements for the
// given refs (peeled entries visible to upload-pack only) and captures the body
// POSTed to /git-receive-pack, responding with the given report-status.
func receivePackServer(
	t *testing.T,
	refs, peeled map[string]string,
	report *packp.ReportStatus,
) (*httptest.Server, *bytes.Buffer, *int) {
	t.Helper()

	uploadAdv := buildAdvertisement(t, "git-upload-pack", refs, peeled, "")
	receiveAdv := buildAdvertisement(t, "git-receive-pack", refs, nil, "")
	postBody := &bytes.Buffer{}
	postCount := new(int)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			*postCount++

			_, err := io.Copy(postBody, r.Body)
			assert.NoError(t, err, "failed to read request body")

			w.WriteHeader(http.StatusOK)
			assert.NoError(t, report.Encode(w), "failed to encode report status")

			return
		}

		adv := receiveAdv
		if r.URL.Query().Get("service") == "git-upload-pack" {
			adv = uploadAdv
		}

		w.WriteHeader(http.StatusOK)

		_, err := w.Write(adv)
		assert.NoError(t, err, "failed to write advertisement")
	}))
	t.Cleanup(s.Close)

	return s, postBody, postCount
}

func okReportStatus() *packp.ReportStatus {
	rs := packp.NewReportStatus()
	rs.UnpackStatus = "ok"
	rs.CommandStatuses = []*packp.CommandStatus{
		{ReferenceName: "refs/heads/feature", Status: "ok"},
	}

	return rs
}

func TestGitProvider_ResolveRemoteReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		wantHash string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name:     "resolves branch name",
			ref:      "master",
			wantHash: testMasterHash,
			wantErr:  require.NoError,
		},
		{
			name:     "resolves another branch name",
			ref:      "branch",
			wantHash: testBranchHash,
			wantErr:  require.NoError,
		},
		{
			name:     "resolves empty reference to HEAD",
			ref:      "",
			wantHash: testMasterHash,
			wantErr:  require.NoError,
		},
		{
			name:     "passes through full commit hash not in advertisement",
			ref:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantErr:  require.NoError,
		},
		{
			name: "returns ErrReferenceNotFound for unknown reference",
			ref:  "no-such-branch",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrReferenceNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := uploadPackServer(t)
			gp := NewGitProvider(Config{Username: "user", Token: "pass"})

			hash, err := gp.ResolveRemoteReference(context.Background(), s.URL, tt.ref)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantHash, hash)
		})
	}
}

func TestGitProvider_CreateRemoteBranchViaRefUpdate(t *testing.T) {
	existingRefs := map[string]string{"refs/heads/master": testMasterHash}

	tests := []struct {
		name      string
		refs      map[string]string
		peeled    map[string]string
		report    *packp.ReportStatus
		branch    string
		fromRef   string
		wantPosts int
		checkBody func(t *testing.T, body []byte)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "creates branch from existing branch with empty packfile",
			refs:      existingRefs,
			report:    okReportStatus(),
			branch:    "feature",
			fromRef:   "master",
			wantPosts: 1,
			checkBody: func(t *testing.T, body []byte) {
				command := plumbing.ZeroHash.String() + " " + testMasterHash + " refs/heads/feature"
				assert.Contains(t, string(body), command, "create command must target the resolved hash")

				pack := emptyPackfileBytes()
				assert.True(t, bytes.HasSuffix(body, pack), "request must end with the empty packfile")
			},
			wantErr: require.NoError,
		},
		{
			name:      "creates branch from full commit hash",
			refs:      existingRefs,
			report:    okReportStatus(),
			branch:    "feature",
			fromRef:   testBranchHash,
			wantPosts: 1,
			checkBody: func(t *testing.T, body []byte) {
				command := plumbing.ZeroHash.String() + " " + testBranchHash + " refs/heads/feature"
				assert.Contains(t, string(body), command)
			},
			wantErr: require.NoError,
		},
		{
			name: "creates branch from annotated tag at the peeled commit",
			refs: map[string]string{
				"refs/heads/master":   testMasterHash,
				"refs/tags/annotated": testTagHash,
			},
			peeled:    map[string]string{"refs/tags/annotated": testPeeledHash},
			report:    okReportStatus(),
			branch:    "feature",
			fromRef:   "annotated",
			wantPosts: 1,
			checkBody: func(t *testing.T, body []byte) {
				command := plumbing.ZeroHash.String() + " " + testPeeledHash + " refs/heads/feature"
				assert.Contains(t, string(body), command,
					"branch must target the peeled commit, not the tag object")
			},
			wantErr: require.NoError,
		},
		{
			name: "skips creation when branch already exists",
			refs: map[string]string{
				"refs/heads/master":  testMasterHash,
				"refs/heads/feature": testBranchHash,
			},
			report:    okReportStatus(),
			branch:    "feature",
			fromRef:   "master",
			wantPosts: 0,
			wantErr:   require.NoError,
		},
		{
			name:      "fails when fromRef cannot be resolved",
			refs:      existingRefs,
			report:    okReportStatus(),
			branch:    "feature",
			fromRef:   "no-such-ref",
			wantPosts: 0,
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrReferenceNotFound)
			},
		},
		{
			name:      "fails when fromRef is empty",
			refs:      existingRefs,
			report:    okReportStatus(),
			branch:    "feature",
			fromRef:   "",
			wantPosts: 0,
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrReferenceNotFound)
			},
		},
		{
			name: "fails when remote rejects the reference update",
			refs: existingRefs,
			report: func() *packp.ReportStatus {
				rs := packp.NewReportStatus()
				rs.UnpackStatus = "ok"
				rs.CommandStatuses = []*packp.CommandStatus{
					{ReferenceName: "refs/heads/feature", Status: "pre-receive hook declined"},
				}

				return rs
			}(),
			branch:    "feature",
			fromRef:   "master",
			wantPosts: 1,
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "pre-receive hook declined", "error must carry the server's rejection reason")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, postBody, postCount := receivePackServer(t, tt.refs, tt.peeled, tt.report)
			gp := NewGitProvider(Config{Username: "user", Token: "pass"})

			err := gp.CreateRemoteBranchViaRefUpdate(context.Background(), s.URL, tt.branch, tt.fromRef)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantPosts, *postCount, "unexpected number of receive-pack POSTs")

			if tt.checkBody != nil {
				tt.checkBody(t, postBody.Bytes())
			}
		})
	}
}

func TestResolveAdvertisedRef(t *testing.T) {
	headHash := plumbing.NewHash(testMasterHash)

	advRefs := packp.NewAdvRefs()
	advRefs.Head = &headHash
	advRefs.References = map[string]plumbing.Hash{
		"refs/heads/master":        plumbing.NewHash(testMasterHash),
		"refs/tags/lightweight":    plumbing.NewHash(testBranchHash),
		"refs/tags/annotated":      plumbing.NewHash(testTagHash),
		"refs/heads/shadowed-name": plumbing.NewHash(testMasterHash),
		"refs/tags/shadowed-name":  plumbing.NewHash(testTagHash),
	}
	advRefs.Peeled = map[string]plumbing.Hash{
		"refs/tags/annotated": plumbing.NewHash(testPeeledHash),
	}

	tests := []struct {
		name     string
		ref      string
		wantHash string
		wantErr  require.ErrorAssertionFunc
	}{
		{name: "branch", ref: "master", wantHash: testMasterHash, wantErr: require.NoError},
		{name: "lightweight tag", ref: "lightweight", wantHash: testBranchHash, wantErr: require.NoError},
		{name: "annotated tag resolves to peeled", ref: "annotated", wantHash: testPeeledHash, wantErr: require.NoError},
		{name: "branch takes precedence over tag", ref: "shadowed-name", wantHash: testMasterHash, wantErr: require.NoError},
		{name: "empty ref resolves HEAD", ref: "", wantHash: testMasterHash, wantErr: require.NoError},
		{name: "full hash passthrough", ref: testPeeledHash, wantHash: testPeeledHash, wantErr: require.NoError},
		{
			name:     "uppercase hash is normalized to lowercase",
			ref:      strings.ToUpper(testPeeledHash),
			wantHash: testPeeledHash,
			wantErr:  require.NoError,
		},
		{
			name: "short hash is not accepted",
			ref:  "6ecf0ef",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrReferenceNotFound)
			},
		},
		{
			name: "unknown ref",
			ref:  "missing",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrReferenceNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := resolveAdvertisedRef(advRefs, tt.ref)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantHash, hash)
		})
	}
}

// An empty repository must resolve to "not found", not a transport failure:
// provisioning probes freshly created projects before the first push.
func TestGitProvider_ResolveRemoteReference_EmptyRepo(t *testing.T) {
	s := emptyUploadPackServer(t)
	gp := NewGitProvider(Config{Username: "user", Token: "pass"})

	_, err := gp.ResolveRemoteReference(context.Background(), s.URL, "main")

	require.ErrorIs(t, err, ErrReferenceNotFound)
}

func TestResolveAdvertisedRef_NoHead(t *testing.T) {
	_, err := resolveAdvertisedRef(packp.NewAdvRefs(), "")

	require.ErrorIs(t, err, ErrReferenceNotFound)
}

func TestEmptyPackfile(t *testing.T) {
	got := emptyPackfileBytes()

	require.Len(t, got, 32, "empty packfile is header (12 bytes) + SHA-1 trailer (20 bytes)")
	assert.Equal(t, []byte("PACK"), got[:4])
	assert.Equal(t, []byte{0, 0, 0, 2}, got[4:8], "packfile version must be 2")
	assert.Equal(t, []byte{0, 0, 0, 0}, got[8:12], "object count must be 0")

	checksum := sha1.Sum(got[:12])
	assert.Equal(t, checksum[:], got[12:], "trailer must be the SHA-1 of the header")
}

func emptyPackfileBytes() []byte {
	pack, err := io.ReadAll(emptyPackfile())
	if err != nil {
		panic(err)
	}

	return pack
}
