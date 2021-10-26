package chain

import (
	"fmt"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gittag/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PushGitTag struct {
	next   handler.GitTagHandler
	client client.Client
	git    gitserver.Git
}

func (h PushGitTag) ServeRequest(gt *v1alpha1.GitTag) error {
	rl := log.WithValues("git tag name", gt.Name)
	rl.Info("start PushGitTag chain executing...")
	if err := h.tryToPushTag(gt); err != nil {
		return errors.Wrapf(err, "couldn't push add tag %v", gt.Spec.Tag)
	}
	rl.Info("end PushGitTag chain executing...")
	return nextServeOrNil(h.next, gt)
}

func (h PushGitTag) tryToPushTag(gt *v1alpha1.GitTag) error {
	c, err := util.GetCodebase(h.client, gt.Spec.Codebase, gt.Namespace)
	if err != nil {
		return err
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, gt.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	if !checkDirectory(wd) {
		ru := fmt.Sprintf("%v:%v", gs.GitHost, *c.Spec.GitUrlPath)
		if err := h.git.CloneRepositoryBySsh(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, ru, wd,
			gs.SshPort); err != nil {
			return err
		}
	}

	err = h.git.Fetch(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, wd, gt.Spec.Branch)
	if err != nil {
		return err
	}

	err = h.git.CreateRemoteTag(string(secret.Data[util.PrivateSShKeyName]), gs.GitUser, wd, gt.Spec.Branch, gt.Spec.Tag)
	if err != nil {
		return err
	}
	return nil
}

func checkDirectory(path string) bool {
	return util.DoesDirectoryExist(path) && !util.IsDirectoryEmpty(path)
}
