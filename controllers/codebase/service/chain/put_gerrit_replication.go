package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
)

type PutGerritReplication struct {
	client client.Client
}

func NewPutGerritReplication(c client.Client) *PutGerritReplication {
	return &PutGerritReplication{client: c}
}

func (h *PutGerritReplication) ServeRequest(_ context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("Start setting Gerrit replication...")

	if err := h.tryToSetupGerritReplication(c.Name, c.Namespace); err != nil {
		setFailedFields(c, codebaseApi.GerritRepositoryProvisioning, err.Error())
		return fmt.Errorf("failed to setup Gerrit replication for codebase %v: %w", c.Name, err)
	}

	rLog.Info("Gerrit replication section finished successfully")

	return nil
}

func (h *PutGerritReplication) tryToSetupGerritReplication(codebaseName, namespace string) error {
	us, err := util.GetUserSettings(h.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get user settings settings: %w", err)
	}

	if !us.VcsIntegrationEnabled {
		log.Info("Skipped Gerrit replication configuration. VCS integration isn't enabled", "codebase_name", codebaseName)
		return nil
	}

	port, err := util.GetGerritPort(h.client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get gerrit port: %w", err)
	}

	vcsConf, err := vcs.GetVcsConfig(h.client, us, codebaseName, namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch VCS config: %w", err)
	}

	s, err := util.GetSecret(h.client, "gerrit-project-creator", namespace)
	if err != nil {
		return fmt.Errorf("failed to get gerrit-project-creator secret: %w", err)
	}

	idrsa := string(s.Data[util.PrivateSShKeyName])
	host := fmt.Sprintf("gerrit.%v", namespace)

	err = gerrit.SetupProjectReplication(h.client, *port, host, idrsa, codebaseName, namespace, vcsConf.VcsSshUrl, log)
	if err != nil {
		return fmt.Errorf("failed to setup project replication: %w", err)
	}

	return nil
}
