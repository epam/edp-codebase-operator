package objectmodifier

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// codebaseModifierFunc is a function that modifies codebase.
type codebaseModifierFunc func(codebase *codebaseApi.Codebase) bool

// CodebaseModifier is a struct that contains a list of codebase modifiers.
type CodebaseModifier struct {
	k8sClient client.Writer
	modifiers []codebaseModifierFunc
}

// NewCodebaseModifier returns a new instance of CodebaseModifier.
func NewCodebaseModifier(k8sClient client.Writer) *CodebaseModifier {
	modifiers := []codebaseModifierFunc{
		trimCodebaseGitSuffix,
		addCodebaseGitSuffix,
	}

	return &CodebaseModifier{k8sClient: k8sClient, modifiers: modifiers}
}

// Apply applies all the codebase modifiers to the codebase.
func (m *CodebaseModifier) Apply(ctx context.Context, codebase *codebaseApi.Codebase) (bool, error) {
	patch := client.MergeFrom(codebase.DeepCopy())
	needToPatch := false

	for _, modifier := range m.modifiers {
		if modifier(codebase) {
			needToPatch = true
		}
	}

	if needToPatch {
		if err := m.k8sClient.Patch(ctx, codebase, patch); err != nil {
			return false, fmt.Errorf("failed to patch codebase: %w", err)
		}

		return true, nil
	}

	return false, nil
}

// trimCodebaseGitSuffix removes all the trailing ".git" suffixes at the end of the git url path, if there are any.
// If it removes anything, it returns true.
func trimCodebaseGitSuffix(codebase *codebaseApi.Codebase) bool {
	if !strings.HasSuffix(codebase.Spec.GitUrlPath, util.CrSuffixGit) {
		return false
	}

	codebase.Spec.GitUrlPath = util.TrimGitFromURL(codebase.Spec.GitUrlPath)

	return true
}

// addCodebaseGitSuffix adds trailing ".git" suffix to the end of the git repository url path, if it doesn't exist.
// Returns true if suffix is added.
func addCodebaseGitSuffix(codebase *codebaseApi.Codebase) bool {
	if codebase.Spec.Strategy != util.CloneStrategy {
		return false
	}

	if codebase.Spec.Repository.Url == "" {
		return false
	}

	if strings.HasSuffix(codebase.Spec.Repository.Url, util.CrSuffixGit) {
		if strings.Count(codebase.Spec.Repository.Url, util.CrSuffixGit) == 1 {
			return false
		}

		codebase.Spec.Repository.Url = util.TrimGitFromURL(codebase.Spec.Repository.Url)
	}

	codebase.Spec.Repository.Url = util.AddGitToURL(codebase.Spec.Repository.Url)

	return true
}
