package gitpathlabel

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

// canonicalHash is the golden value for "/myorg/my-app" and every case/slash
// variant that must converge to it.
const canonicalHash = "89f3172369ae09a6b91fa78494b89639"

// TestHashLengthFrozen guards the wire format independently of the golden
// vectors: changing hashLen must fail an explicit assertion, not just require
// regenerating golden strings in the same file.
func TestHashLengthFrozen(t *testing.T) {
	if hashLen != 32 {
		t.Fatalf("hashLen = %d, must stay 32: the value is persisted in labels cluster-wide "+
			"and recomputed independently by edp-tekton", hashLen)
	}
}

// TestHashGolden pins the wire format: these values are persisted in labels
// cluster-wide and recomputed independently by edp-tekton. If this test fails,
// do not update the vectors — revert the change, it breaks every existing label.
func TestHashGolden(t *testing.T) {
	tests := []struct {
		name       string
		gitUrlPath string
		want       string
	}{
		{
			name:       "canonical form",
			gitUrlPath: "/myorg/my-app",
			want:       canonicalHash,
		},
		{
			name:       "mixed case converges",
			gitUrlPath: "/MyOrg/My-App",
			want:       canonicalHash,
		},
		{
			name:       "missing leading slash converges",
			gitUrlPath: "myorg/my-app",
			want:       canonicalHash,
		},
		{
			name:       "upper case converges",
			gitUrlPath: "/MYORG/MY-APP",
			want:       canonicalHash,
		},
		{
			// Mirrors EqualFold semantics: a trailing slash is a different identity.
			name:       "trailing slash is a different identity",
			gitUrlPath: "/myorg/my-app/",
			want:       "ccf6c216ffae94a2d7a77d9f4546febc",
		},
		{
			name:       "empty path normalizes to /",
			gitUrlPath: "",
			want:       "8a5edab282632443219e051e4ade2d1d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Hash(tt.gitUrlPath); got != tt.want {
				t.Fatalf("Hash(%q) = %q, want %q", tt.gitUrlPath, got, tt.want)
			}
		})
	}
}

func TestHashIsValidLabelValue(t *testing.T) {
	inputs := []string{"/myorg/my-app", "", "/UPPER/Case", "no-slash", "/very/deep/group/nesting/repository-name"}
	for _, in := range inputs {
		if errs := validation.IsValidLabelValue(Hash(in)); len(errs) != 0 {
			t.Fatalf("Hash(%q) is not a valid label value: %v", in, errs)
		}
	}
}
