package chain

import (
	"context"
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const gerritQuickLinkName = "gerrit"

// PutGitWebRepoUrl is a chain element that puts GitWebUrlPath for codebase status subresource.
type PutGitWebRepoUrl struct {
	client client.Client
}

// NewPutGitWebRepoUrl creates a new PutGitWebRepoUrl chain element.
func NewPutGitWebRepoUrl(k8sClient client.Client) *PutGitWebRepoUrl {
	return &PutGitWebRepoUrl{client: k8sClient}
}

// ServeRequest put Git Repo URL Path.
func (s *PutGitWebRepoUrl) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start putting urlRepoPath to codebase status")

	gitServer := &codebaseApi.GitServer{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace}, gitServer); err != nil {
		return s.processCodebaseError(
			codebase,
			fmt.Errorf("failed to get git server %s: %w", codebase.Spec.GitServer, err),
		)
	}

	gitWebURL, err := s.getGitWebURL(ctx, gitServer, codebase)
	if err != nil {
		return s.processCodebaseError(codebase, err)
	}

	codebase.Status.GitWebUrl = gitWebURL

	if err = setIntermediateSuccessFields(ctx, s.client, codebase, codebaseApi.PutGitWebRepoUrl); err != nil {
		return fmt.Errorf("failed to update codebase %s status: %w", codebase.Name, err)
	}

	log.Info("Finish putting urlRepoPath to codebase status")

	return nil
}

// getGitWebURL returns Git Web URL.
// For GitHub and GitLab we return link to the repository in format: https://<git_host>/<git_org>/<git_repo>
// For Gerrit we return link to the repository in format: https://<gerrit_host>/gitweb?p=<codebase>.git
func (s *PutGitWebRepoUrl) getGitWebURL(ctx context.Context, gitServer *codebaseApi.GitServer, codebase *codebaseApi.Codebase) (string, error) {
	switch gitServer.Spec.GitProvider {
	case codebaseApi.GitProviderGitlab, codebaseApi.GitProviderGithub:
		urlLink := util.GetHostWithProtocol(gitServer.Spec.GitHost)
		urlLink = strings.TrimSuffix(urlLink, "/")
		// For GitHub and GitLab we return link to the repository in format: https://<git_host>/<git_org>/<git_repo>
		return fmt.Sprintf("%s/%s", urlLink, codebase.Spec.GetProjectID()), nil

	case codebaseApi.GitProviderGerrit:
		link := &codebaseApi.QuickLink{}
		if err := s.client.Get(ctx, client.ObjectKey{Name: gerritQuickLinkName, Namespace: gitServer.Namespace}, link); err != nil {
			return "", fmt.Errorf("failed to get QuickLink %s: %w", gerritQuickLinkName, err)
		}
		// QuickLink has https:// prefix in the URL.
		// For Gerrit, we return link to the repository in format: https://<gerrit_host>/gitweb?p=<codebase>.git
		gerritProjectUrl := strings.TrimSuffix(link.Spec.Url, "/")

		return fmt.Sprintf("%s/gitweb?p=%s.git", gerritProjectUrl, codebase.Spec.GetProjectID()), nil
	default:
		return "", fmt.Errorf("unsupported Git provider %s", gitServer.Spec.GitProvider)
	}
}

func (*PutGitWebRepoUrl) processCodebaseError(codebase *codebaseApi.Codebase, err error) error {
	setFailedFields(codebase, codebaseApi.PutGitWebRepoUrl, err.Error())

	return err
}
