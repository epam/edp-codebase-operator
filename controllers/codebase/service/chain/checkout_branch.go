package chain

import (
	"fmt"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	git "github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
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

func CheckoutBranch(repository *string, projectPath, branchName string, g git.Git, cb *codebaseApi.Codebase, c client.Client) error {
	user, password, err := GetRepositoryCredentialsIfExists(cb, c)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}

	if !g.CheckPermissions(*repository, user, password) {
		msg := fmt.Errorf("user %s cannot get access to the repository %s", *user, *repository)
		return msg
	}

	currentBranchName, err := g.GetCurrentBranchName(projectPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch name: %w", err)
	}

	if currentBranchName == branchName {
		log.Info("default branch is already active", "name", branchName)
		return nil
	}

	switch cb.Spec.Strategy {
	case "create":
		if err := g.Checkout(user, password, projectPath, branchName, false); err != nil {
			return errors.Wrapf(err, "checkout default branch %s has been failed (create strategy)",
				branchName)
		}

	case "clone":
		if err := g.Checkout(user, password, projectPath, branchName, true); err != nil {
			return errors.Wrapf(err, "checkout default branch %s has been failed (clone strategy)",
				branchName)
		}
	case "import":
		gs, err := util.GetGitServer(c, cb.Spec.GitServer, cb.Namespace)
		if err != nil {
			return errors.Wrapf(err, "Unable to get GitServer")
		}

		secret, err := util.GetSecret(c, gs.NameSshKeySecret, cb.Namespace)
		if err != nil {
			return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
		}

		k := string(secret.Data[util.PrivateSShKeyName])
		u := gs.GitUser
		// CheckoutRemoteBranchBySSH(key, user, gitPath, remoteBranchName string)
		if err := g.CheckoutRemoteBranchBySSH(k, u, projectPath, branchName); err != nil {
			return errors.Wrapf(err, "checkout default branch %s has been failed (import strategy)",
				branchName)
		}

	default:
		return fmt.Errorf("unable to checkout, unsupported strategy: '%s'", cb.Spec.Strategy)
	}

	return nil
}
