package chain

import (
	"fmt"

	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutGerritReplication struct {
	next   handler.CodebaseHandler
	client client.Client
}

func (h PutGerritReplication) ServeRequest(c *edpv1alpha1.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("Start setting Gerrit replication...")

	if err := h.tryToSetupGerritReplication(c.Name, c.Namespace); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "setup Gerrit replication for codebase %v has been failed", c.Name)
	}
	rLog.Info("Gerrit replication section finished successfully")
	return nextServeOrNil(h.next, c)
}

func (h PutGerritReplication) tryToSetupGerritReplication(codebaseName, namespace string) error {
	us, err := util.GetUserSettings(h.client, namespace)
	if err != nil {
		return errors.Wrap(err, "unable get user settings settings")
	}

	if !us.VcsIntegrationEnabled {
		log.Info("Skipped Gerrit replication configuration. VCS integration isn't enabled", "codebase_name", codebaseName)
		return nil
	}

	port, err := util.GetGerritPort(h.client, namespace)
	if err != nil {
		return errors.Wrap(err, "unable get gerrit port")
	}

	vcsConf, err := vcs.GetVcsConfig(h.client, us, codebaseName, namespace)
	if err != nil {
		return err
	}

	s, err := util.GetSecret(h.client, "gerrit-project-creator", namespace)
	if err != nil {
		return errors.Wrap(err, "unable to get gerrit-project-creator secret")
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	host := fmt.Sprintf("gerrit.%v", namespace)
	return gerrit.SetupProjectReplication(h.client, *port, host, idrsa, codebaseName, namespace, vcsConf.VcsSshUrl,
		log)
}
