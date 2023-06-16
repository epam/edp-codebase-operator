package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/gittag/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PushGitTag struct {
	next   handler.GitTagHandler
	client client.Client
	git    git.Git
}

func (h PushGitTag) ServeRequest(gt *codebaseApi.GitTag) error {
	rl := log.WithValues("git tag name", gt.Name)

	rl.Info("start PushGitTag chain executing...")

	if err := h.tryToPushTag(gt); err != nil {
		return fmt.Errorf("failed to push add tag %v: %w", gt.Spec.Tag, err)
	}

	rl.Info("end PushGitTag chain executing...")

	err := nextServeOrNil(h.next, gt)
	if err != nil {
		return fmt.Errorf("failed to process next handler in a chain: %w", err)
	}

	return nil
}

func (h PushGitTag) tryToPushTag(gt *codebaseApi.GitTag) error {
	c, err := util.GetCodebase(h.client, gt.Spec.Codebase, gt.Namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch Codebase: %w", err)
	}

	gitServer := &codebaseApi.GitServer{}
	if err = h.client.Get(
		context.TODO(),
		client.ObjectKey{
			Namespace: gt.Namespace,
			Name:      c.Spec.GitServer,
		},
		gitServer,
	); err != nil {
		return fmt.Errorf("failed to fetch Git Server: %w", err)
	}

	secret, err := util.GetSecret(h.client, gitServer.Spec.NameSshKeySecret, c.Namespace)
	if err != nil {
		return fmt.Errorf("an error has occurred while getting %v secret: %w", gitServer.Spec.NameSshKeySecret, err)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	if !checkDirectory(wd) {
		repoSshUrl := util.GetSSHUrl(gitServer, c.Spec.GetProjectID())
		key := string(secret.Data[util.PrivateSShKeyName])

		err = h.git.CloneRepositoryBySsh(context.TODO(), key, gitServer.Spec.GitUser, repoSshUrl, wd, gitServer.Spec.SshPort)
		if err != nil {
			return fmt.Errorf("failed to git cline repository: %w", err)
		}
	}

	err = h.git.Fetch(string(secret.Data[util.PrivateSShKeyName]), gitServer.Spec.GitUser, wd, gt.Spec.Branch)
	if err != nil {
		return fmt.Errorf("failed to git fetch: %w", err)
	}

	err = h.git.CreateRemoteTag(string(secret.Data[util.PrivateSShKeyName]), gitServer.Spec.GitUser, wd, gt.Spec.Branch, gt.Spec.Tag)
	if err != nil {
		return fmt.Errorf("failed to create remote tag in git: %w", err)
	}

	return nil
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}
