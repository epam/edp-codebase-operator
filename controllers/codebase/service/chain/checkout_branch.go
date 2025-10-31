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
	repository, projectPath, branchName string,
	g gitproviderv2.Git,
	cb *codebaseApi.Codebase,
	c client.Client,
	createGitProviderWithConfig func(config gitproviderv2.Config) gitproviderv2.Git,
) error {
	currentBranchName, err := g.GetCurrentBranchName(ctx, projectPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch name: %w", err)
	}

	if currentBranchName == branchName {
		ctrl.Log.Info("default branch is already active", "name", branchName)
		return nil
	}

	switch cb.Spec.Strategy {
	case "create":
		if err := g.Checkout(ctx, projectPath, branchName, false); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s (create strategy): %w", branchName, err)
		}

	case "clone":
		user, password, err := GetRepositoryCredentialsIfExists(cb, c)
		if err != nil && !k8sErrors.IsNotFound(err) {
			return err
		}

		cloneRepoGitProvider := g

		if user != nil && password != nil {
			cloneRepoGitProvider = createGitProviderWithConfig(gitproviderv2.Config{
				Username: *user,
				Token:    *password,
			})
		}

		if err := cloneRepoGitProvider.Checkout(ctx, projectPath, branchName, true); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s (clone strategy): %w", branchName, err)
		}
	case "import":
		if err := g.CheckoutRemoteBranch(ctx, projectPath, branchName); err != nil {
			return fmt.Errorf("failed to checkout to default branch %s (import strategy): %w", branchName, err)
		}
	default:
		return fmt.Errorf("failed to checkout, unsupported strategy: '%s'", cb.Spec.Strategy)
	}

	return nil
}
