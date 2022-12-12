package chain

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/controllers/gittag/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PushGitTag struct {
	next   handler.GitTagHandler
	client client.Client
	git    gitserver.Git
}

func (h PushGitTag) ServeRequest(gt *codebaseApi.GitTag) error {
	rl := log.WithValues("git tag name", gt.Name)

	rl.Info("start PushGitTag chain executing...")

	if err := h.tryToPushTag(gt); err != nil {
		return errors.Wrapf(err, "couldn't push add tag %v", gt.Spec.Tag)
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

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, gt.Namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch Git Server: %w", err)
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	if !checkDirectory(wd) {
		ru := fmt.Sprintf("%v:%v", gs.GitHost, *c.Spec.GitUrlPath)
		key := string(secret.Data[util.PrivateSShKeyName])

		err = h.git.CloneRepositoryBySsh(key, gs.GitUser, ru, wd, gs.SshPort)
		if err != nil {
			return fmt.Errorf("failed to git cline repository: %w", err)
		}
	}

	err = h.git.Fetch(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, wd, gt.Spec.Branch)
	if err != nil {
		return fmt.Errorf("failed to git fetch: %w", err)
	}

	err = h.git.CreateRemoteTag(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, wd, gt.Spec.Branch, gt.Spec.Tag)
	if err != nil {
		return fmt.Errorf("failed to create remote tag in git: %w", err)
	}

	return nil
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}
