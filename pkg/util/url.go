package util

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	logCodebaseNameKey = "codebase_name"
	CrSuffixGit        = ".git"
)

var (
	protocolRegexp = regexp.MustCompile(`^(https://)|^(http://)`)
)

func TrimGitFromURL(url string) string {
	for strings.HasSuffix(url, CrSuffixGit) {
		url = strings.TrimSuffix(url, CrSuffixGit)
	}

	return url
}

func AddGitToURL(url string) string {
	if !strings.HasSuffix(url, CrSuffixGit) {
		url += CrSuffixGit
	}

	return url
}

func GetRepoUrl(c *codebaseApi.Codebase) (string, error) {
	log.Info("Setup repo url", "codebase_name", c.Name)

	if c.Spec.Strategy == codebaseApi.Clone {
		log.Info("strategy is clone. Try to use default value...", logCodebaseNameKey, c.Name)
		return tryGetRepoUrl(&c.Spec)
	}

	log.Info("TriggerType is not clone. Start build url...", logCodebaseNameKey, c.Name)

	u := BuildRepoUrl(&c.Spec)

	log.Info("Repository url has been generated", "url", u, logCodebaseNameKey, c.Name)

	return u, nil
}

func tryGetRepoUrl(spec *codebaseApi.CodebaseSpec) (string, error) {
	if spec.Repository == nil {
		return "", errors.New("repository cannot be nil for specified strategy")
	}

	return spec.Repository.Url, nil
}

func BuildRepoUrl(spec *codebaseApi.CodebaseSpec) string {
	log.Info("Start building repo url", "base url", GithubDomain, "spec", spec)

	return strings.ToLower(
		fmt.Sprintf(
			"%v/%v-%v-%v.git",
			GithubDomain,
			spec.Lang, spec.BuildTool,
			spec.Framework,
		),
	)
}

// GetHostWithProtocol adds protocol to host if it is not presented.
func GetHostWithProtocol(host string) string {
	if protocolRegexp.MatchString(host) {
		return host
	}

	return fmt.Sprintf("https://%v", host)
}

// GetSSHUrl returns ssh url for git server.
func GetSSHUrl(gitServer *codebaseApi.GitServer, repoName string) string {
	if gitServer.Spec.GitProvider == codebaseApi.GitProviderGerrit {
		return fmt.Sprintf("ssh://%s:%d/%s", gitServer.Spec.GitHost, gitServer.Spec.SshPort, repoName)
	}

	return fmt.Sprintf("git@%s:%s.git", gitServer.Spec.GitHost, repoName)
}
