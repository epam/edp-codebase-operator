package chain

import (
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
        k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRepositoryCredentialsIfExists(c *v1alpha1.Codebase, client client.Client) (*string, *string, error) {
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

func CheckoutBranch(repository *string, projectPath, branchName string, git git.Git, c *v1alpha1.Codebase, client client.Client) error {
	user, password, err := GetRepositoryCredentialsIfExists(c, client)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if !git.CheckPermissions(*repository, user, password) {
		msg := fmt.Errorf("user %v cannot get access to the repository %v", user, *repository)
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

	remote := true
	if c.Spec.Strategy != "Create" {
		remote = false
	}

	if err := git.Checkout(user, password, projectPath, branchName, remote); err != nil {
		return errors.Wrapf(err, "checkout default branch %v has been failed", branchName)
	}
	return nil
}