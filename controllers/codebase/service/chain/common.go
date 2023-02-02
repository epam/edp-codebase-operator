package chain

import (
	"context"
	"fmt"
	"regexp"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var protocolRegexp = regexp.MustCompile(`^(https://)|^(http://)`)

func pushChangesToGit(c client.Client, g git.Git, projectPath string, cb *codebaseApi.Codebase) error {
	gs, err := util.GetGitServer(c, cb.Spec.GitServer, cb.Namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch GitServer: %w", err)
	}

	secret, err := util.GetSecret(c, gs.NameSshKeySecret, cb.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get %v secret: %w", gs.NameSshKeySecret, err)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	p := gs.SshPort

	if err := g.PushChanges(k, u, projectPath, p, cb.Spec.DefaultBranch); err != nil {
		return fmt.Errorf("failed to push changes for repo %v: %w", projectPath, err)
	}

	log.Info("templates have been pushed")

	return nil
}

func setIntermediateSuccessFields(ctx context.Context, c client.Client, cb *codebaseApi.Codebase, action codebaseApi.ActionType) error {
	cb.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Action:          action,
		Result:          codebaseApi.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    cb.Status.FailureCount,
		Git:             cb.Status.Git,
		WebHookID:       cb.Status.WebHookID,
	}

	err := c.Status().Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update status field of %q resource 'Codebase': %w", cb.Name, err)
	}

	err = c.Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update %q resource 'Codebase': %w", cb.Name, err)
	}

	return nil
}

func getHostWithProtocol(host string) string {
	if protocolRegexp.MatchString(host) {
		return host
	}

	return fmt.Sprintf("https://%v", host)
}

// getGitProviderAPIURL returns git server url with protocol.
func getGitProviderAPIURL(gitServer *codebaseApi.GitServer) string {
	url := getHostWithProtocol(gitServer.Spec.GitHost)

	if gitServer.Spec.GitProvider == codebaseApi.GitProviderGithub {
		// GitHub API url is different for enterprise and other versions
		// see: https://docs.github.com/en/get-started/learning-about-github/about-versions-of-github-docs#github-enterprise-server
		if url == "https://github.com" {
			return "https://api.github.com"
		}

		url = fmt.Sprintf("%s/api/v3", url)
	}

	if gitServer.Spec.HttpsPort != 0 {
		url = fmt.Sprintf("%s:%d", url, gitServer.Spec.HttpsPort)
	}

	return url
}
