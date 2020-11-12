package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	perfAPi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		log.Info("PERF server wasn't selected. skip creating PERF data source cr...",
			"codebase name", c.Name)
		return nil
	}

	for _, name := range c.Spec.Perf.DataSources {
		if err := h.tryToCreateDataSource(c, name); err != nil {
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
			JobNames: []string{fmt.Sprintf("/%v/%v-Build-%v", codebase, strings.ToUpper(branch), codebase)},
			Url:      c.Spec.Url,
		}, nil
	}
	c, err := util.GetEdpComponent(h.client, sonarEdpComponentName, namespace)
	if err != nil {
		return nil, err
	}

	return &perfAPi.DataSourceConfig{
		ProjectKeys: []string{fmt.Sprintf("%v", codebase)},
		Url:         c.Spec.Url,
	}, nil
}

func (h PutPerfDataSources) tryToCreateDataSource(c *v1alpha1.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSource{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      getDataSourceName(c.Name, dataSourceType),
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			config, err := h.getDataSourceConfig(dataSourceType, c.Spec.DefaultBranch, c.Name, c.Namespace)
			if err != nil {
				return err
			}
			return h.createDataSource(c, dataSourceType, *config)
		}
		return err
	}
	return nil
}

func (h PutPerfDataSources) createDataSource(c *v1alpha1.Codebase, dataSourceType string, config perfAPi.DataSourceConfig) error {
	ds := &perfAPi.PerfDataSource{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "PerfDataSource",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      getDataSourceName(c.Name, dataSourceType),
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

func getDataSourceName(codebase, dataSourceType string) string {
	return strings.ToLower(fmt.Sprintf("%v-%v", codebase, dataSourceType))
}
