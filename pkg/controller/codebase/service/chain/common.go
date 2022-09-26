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

func pushChangesToGit(c client.Client, g git.Git, projectPath string, cb *codebaseApi.Codebase) error {
	gs, err := util.GetGitServer(c, cb.Spec.GitServer, cb.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(c, gs.NameSshKeySecret, cb.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	p := gs.SshPort
	if err := g.PushChanges(k, u, projectPath, p, cb.Spec.DefaultBranch); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v repo", projectPath)
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
	}

	if err := c.Status().Update(ctx, cb); err != nil {
		if err := c.Update(ctx, cb); err != nil {
			return err
		}
	}
	return nil
}
