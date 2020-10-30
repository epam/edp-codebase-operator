package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	perfAPi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type PutPerfDataSources struct {
	next   handler.CodebaseHandler
	client client.Client
}

const (
	jenkinsEdpComponentName = "jenkins"
	sonarEdpComponentName   = "sonar"
	defaultBranch           = "master"

	jenkinsDataSourceSecretName = "jenkins-admin-token"
	sonarDataSourceSecretName   = "sonar-admin-password"
)

func (h PutPerfDataSources) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("start creating PERF data source cr...")
	if err := h.tryToCreateDataSourceCr(c); err != nil {
		return errors.Wrap(err, "couldn't create PerfDataSource CR")
	}
	rLog.Info("data source has been created")
	return nextServeOrNil(h.next, c)
}

func (h PutPerfDataSources) tryToCreateDataSourceCr(c *v1alpha1.Codebase) error {
	if c.Spec.Perf == nil {
		log.Info("PERF server wasn't selected. skip creating PERF data source cr...")
		return nil
	}

	for _, name := range c.Spec.Perf.DataSources {
		config, err := h.getDataSourceConfig(name, defaultBranch, c.Name, c.Namespace)
		if err != nil {
			return err
		}

		if err := h.createDataSource(c, name, *config); err != nil {
			return err
		}
	}
	return nil
}

func (h PutPerfDataSources) getDataSourceConfig(dsType, branch, codebase, namespace string) (*perfAPi.DataSourceConfig, error) {
	if dsType == "Jenkins" {
		c, err := util.GetEdpComponent(h.client, jenkinsEdpComponentName, namespace)
		if err != nil {
			return nil, err
		}

		return &perfAPi.DataSourceConfig{
			JobNames:       []string{fmt.Sprintf("/%v/%v-Build-%v", codebase, strings.ToUpper(branch), codebase)},
			Url:            c.Spec.Url,
			CredentialName: jenkinsDataSourceSecretName,
		}, nil
	}
	c, err := util.GetEdpComponent(h.client, sonarEdpComponentName, namespace)
	if err != nil {
		return nil, err
	}

	return &perfAPi.DataSourceConfig{
		ProjectKeys:    []string{fmt.Sprintf("%v", codebase)},
		Url:            c.Spec.Url,
		CredentialName: sonarDataSourceSecretName,
	}, nil
}

func (h PutPerfDataSources) createDataSource(c *v1alpha1.Codebase, dataSourceType string, config perfAPi.DataSourceConfig) error {
	ds := &perfAPi.PerfDataSource{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "PerfDataSource",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      strings.ToLower(fmt.Sprintf("%v-%v", c.Name, dataSourceType)),
			Namespace: c.Namespace,
		},
		Spec: perfAPi.PerfDataSourceSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         config,
			PerfServerName: c.Spec.Perf.Name,
		},
	}

	if err := h.client.Create(context.TODO(), ds); err != nil {
		return errors.Wrapf(err, "couldn't create PERF data source %v-%v", c.Name, dataSourceType)
	}
	return nil
}
