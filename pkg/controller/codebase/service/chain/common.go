package chain

import (
	"context"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func pushChangesToGit(client client.Client, git git.Git, projectPath string, c *codebaseApi.Codebase) error {
	gs, err := util.GetGitServer(client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := git.PushChanges(k, u, projectPath, c.Spec.DefaultBranch); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v repo", projectPath)
	}
	log.Info("templates have been pushed")
	return nil
}

func setIntermediateSuccessFields(ctx context.Context, client client.Client, c *codebaseApi.Codebase, action codebaseApi.ActionType) error {
	c.Status = codebaseApi.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Action:          action,
		Result:          codebaseApi.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    c.Status.FailureCount,
		Git:             c.Status.Git,
	}

	if err := client.Status().Update(ctx, c); err != nil {
		if err := client.Update(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
