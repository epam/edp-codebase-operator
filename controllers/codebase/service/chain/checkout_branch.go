package chain

import (
	"context"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func GetRepositoryCredentialsIfExists(cb *codebaseApi.Codebase, c client.Client) (userName, password *string, err error) {
	if cb.Spec.Repository == nil {
		return nil, nil, nil
	}

	secret := fmt.Sprintf("repository-codebase-%v-temp", cb.Name)

	repositoryUsername, repositoryPassword, err := util.GetVcsBasicAuthConfig(c, cb.Namespace, secret)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch VCS auth config: %w", err)
	}

	userName = &repositoryUsername
	password = &repositoryPassword

	return
}

func CheckoutBranch(
	ctx context.Context,
	branchName string,
	repoContext *GitRepositoryContext,
	cb *codebaseApi.Codebase,
	c client.Client,
	gitProviderFactory func(config gitproviderv2.Config) gitproviderv2.Git,
) error {
	log := ctrl.LoggerFrom(ctx)
	gitProvider := gitProviderFactory(gitproviderv2.NewConfigFromGitServerAndSecret(repoContext.GitServer, repoContext.GitServerSecret))

	currentBranchName, err := gitProvider.GetCurrentBranchName(ctx, repoContext.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get current branch name: %w", err)
	}

	if currentBranchName == branchName {
		log.Info("Default branch is already active", "name", branchName)
		return nil
	}

	switch cb.Spec.Strategy {
	case "create":
		if err := gitProvider.Checkout(ctx, repoContext.WorkDir, branchName, false); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s (create strategy): %w", branchName, err)
		}

	case "clone":
		user, password, err := GetRepositoryCredentialsIfExists(cb, c)
		if err != nil && !k8sErrors.IsNotFound(err) {
			return err
		}

		cfg := gitproviderv2.Config{}
		if user != nil && password != nil {
			cfg.Username = *user
			cfg.Token = *password
		}

		if err := gitProviderFactory(cfg).Checkout(ctx, repoContext.WorkDir, branchName, true); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s (clone strategy): %w", branchName, err)
		}
	case "import":
		if err := gitProvider.CheckoutRemoteBranch(ctx, repoContext.WorkDir, branchName); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s (import strategy): %w", branchName, err)
		}
	default:
		return fmt.Errorf("failed to checkout, unsupported strategy: '%s'", cb.Spec.Strategy)
	}

	return nil
}
