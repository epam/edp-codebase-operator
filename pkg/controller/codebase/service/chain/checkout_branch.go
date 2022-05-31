package chain

import (
	"fmt"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func GetRepositoryCredentialsIfExists(c *codebaseApi.Codebase, client client.Client) (*string, *string, error) {
	if c.Spec.Repository == nil {
		return nil, nil, nil
	}
	secret := fmt.Sprintf("repository-codebase-%v-temp", c.Name)
	repositoryUsername, repositoryPassword, err := util.GetVcsBasicAuthConfig(client, c.Namespace, secret)
	if err != nil {
		return nil, nil, err
	}
	return &repositoryUsername, &repositoryPassword, nil
}

func CheckoutBranch(repository *string, projectPath, branchName string, git git.Git, c *codebaseApi.Codebase, client client.Client) error {
	user, password, err := GetRepositoryCredentialsIfExists(c, client)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	if !git.CheckPermissions(*repository, user, password) {
		msg := fmt.Errorf("user %s cannot get access to the repository %s", *user, *repository)
		return msg
	}
	currentBranchName, err := git.GetCurrentBranchName(projectPath)
	if err != nil {
		return err
	}
	if currentBranchName == branchName {
		log.Info("default branch is already active", "name", branchName)
		return nil
	}

	switch c.Spec.Strategy {
	case "create":
		if err := git.Checkout(user, password, projectPath, branchName, false); err != nil {
			return errors.Wrapf(err, "checkout default branch %s has been failed (create strategy)",
				branchName)
		}

	case "clone":
		if err := git.Checkout(user, password, projectPath, branchName, true); err != nil {
			return errors.Wrapf(err, "checkout default branch %s has been failed (clone strategy)",
				branchName)
		}
	case "import":
		gs, err := util.GetGitServer(client, c.Spec.GitServer, c.Namespace)
		if err != nil {
			return errors.Wrapf(err, "Unable to get GitServer")
		}

		secret, err := util.GetSecret(client, gs.NameSshKeySecret, c.Namespace)
		if err != nil {
			return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
		}

		k := string(secret.Data[util.PrivateSShKeyName])
		u := gs.GitUser
		// CheckoutRemoteBranchBySSH(key, user, gitPath, remoteBranchName string)
		if err := git.CheckoutRemoteBranchBySSH(k, u, projectPath, branchName); err != nil {
			return errors.Wrapf(err, "checkout default branch %s has been failed (import strategy)",
				branchName)
		}

	default:
		return fmt.Errorf("unable to checkout, unsupported strategy: '%s'", c.Spec.Strategy)
	}

	return nil
}
