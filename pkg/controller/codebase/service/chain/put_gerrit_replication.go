package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/gerrit"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/codebase-operator/v2/pkg/vcs"
	"github.com/pkg/errors"
)

type PutGerritReplication struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h PutGerritReplication) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start setting Gerrit replication...")

	if err := h.tryToSetupGerritReplication(c.Name, c.Namespace); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "setup Gerrit replication for codebase %v has been failed", c.Name)
	}
	log.Info("Gerrit configuration has been finished successfully")
	return nextServeOrNil(h.next, c)
}

func (h PutGerritReplication) tryToSetupGerritReplication(codebaseName, namespace string) error {
	log.Info("Start setting Gerrit replication", "codebase name", codebaseName)

	gs, us, err := util.GetConfigSettings(h.clientSet.CoreClient, namespace)
	if err != nil {
		return errors.Wrap(err, "unable get config settings")
	}

	if us.VcsIntegrationEnabled {
		vcsConf, err := vcs.GetVcsConfig(*h.clientSet.CoreClient, us, codebaseName, namespace)
		if err != nil {
			return err
		}

		s, err := util.GetSecret(*h.clientSet.CoreClient, "gerrit-project-creator", namespace)
		if err != nil {
			return errors.Wrap(err, "unable to get gerrit-project-creator secret")
		}

		idrsa := string(s.Data[util.PrivateSShKeyName])
		host := fmt.Sprintf("gerrit.%v", namespace)
		return gerrit.SetupProjectReplication(*h.clientSet.CoreClient, gs.SshPort, host, idrsa, codebaseName, vcsConf.VcsSshUrl, namespace)
	}
	log.Info("Skipped Gerrit replication configuration. VCS integration isn't enabled")
	return nil
}
