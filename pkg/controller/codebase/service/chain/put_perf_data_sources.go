package chain

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	perfAPi "github.com/epam/edp-perf-operator/v2/pkg/apis/edp/v1alpha1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutPerfDataSources struct {
	next   handler.CodebaseHandler
	client client.Client
}

const (
	jenkinsEdpComponentName = "jenkins"
	sonarEdpComponentName   = "sonar"

	jenkinsDataSourceType = "Jenkins"
	sonarDataSourceType   = "Sonar"
	gitLabDataSourceType  = "GitLab"
)

func (h PutPerfDataSources) ServeRequest(c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("start creating PERF data source cr...")
	if err := h.tryToCreateDataSourceCr(c); err != nil {
		return errors.Wrap(err, "couldn't create PerfDataSource CR")
	}
	rLog.Info("data source has been created")
	return nextServeOrNil(h.next, c)
}

func (h PutPerfDataSources) tryToCreateDataSourceCr(c *codebaseApi.Codebase) error {
	if c.Spec.Perf == nil {
		log.Info("PERF server wasn't selected. skip creating PERF data source cr...",
			"codebase_name", c.Name)
		return nil
	}

	createFactory := h.getCreateFactory()
	for _, name := range c.Spec.Perf.DataSources {
		if err := createFactory[name](c, name); err != nil {
			return err
		}
	}
	return nil
}

func (h PutPerfDataSources) getCreateFactory() map[string]func(c *codebaseApi.Codebase, dataSourceType string) error {
	return map[string]func(c *codebaseApi.Codebase, dataSourceType string) error{
		jenkinsDataSourceType: h.tryToCreateJenkinsDataSource,
		sonarDataSourceType:   h.tryToCreateSonarDataSource,
		gitLabDataSourceType:  h.tryToCreateGitLabDataSource,
	}
}

func (h PutPerfDataSources) tryToCreateJenkinsDataSource(c *codebaseApi.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSourceJenkins{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      getDataSourceName(c.Name, dataSourceType),
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return h.createJenkinsDataSource(c, dataSourceType)
		}
		return err
	}
	return nil
}

func (h PutPerfDataSources) tryToCreateSonarDataSource(c *codebaseApi.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSourceSonar{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      getDataSourceName(c.Name, dataSourceType),
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return h.createSonarDataSource(c, dataSourceType)
		}
		return err
	}
	return nil
}

func (h PutPerfDataSources) tryToCreateGitLabDataSource(c *codebaseApi.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSourceGitLab{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      getDataSourceName(c.Name, dataSourceType),
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return h.createGitLabDataSource(c, dataSourceType)
		}
		return err
	}
	return nil
}

func (h PutPerfDataSources) createJenkinsDataSource(c *codebaseApi.Codebase, dataSourceType string) error {
	config, err := h.getJenkinsDataSourceConfig(c.Spec.DefaultBranch, c.Name, c.Namespace)
	if err != nil {
		return err
	}

	ds := &perfAPi.PerfDataSourceJenkins{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "PerfDataSourceJenkins",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      getDataSourceName(c.Name, dataSourceType),
			Namespace: c.Namespace,
		},
		Spec: perfAPi.PerfDataSourceJenkinsSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         *config,
			PerfServerName: c.Spec.Perf.Name,
			CodebaseName:   c.Name,
		},
	}

	if err := h.client.Create(context.TODO(), ds); err != nil {
		return errors.Wrapf(err, "couldn't create PERF Jenkins data source %v-%v", c.Name, dataSourceType)
	}
	return nil
}

func (h PutPerfDataSources) createSonarDataSource(c *codebaseApi.Codebase, dataSourceType string) error {
	config, err := h.getSonarDataSourceConfig(c.Spec.DefaultBranch, c.Name, c.Namespace)
	if err != nil {
		return err
	}

	ds := &perfAPi.PerfDataSourceSonar{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "PerfDataSourceSonar",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      getDataSourceName(c.Name, dataSourceType),
			Namespace: c.Namespace,
		},
		Spec: perfAPi.PerfDataSourceSonarSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         *config,
			PerfServerName: c.Spec.Perf.Name,
			CodebaseName:   c.Name,
		},
	}

	if err := h.client.Create(context.TODO(), ds); err != nil {
		return errors.Wrapf(err, "couldn't create PERF Sonar data source %v-%v", c.Name, dataSourceType)
	}
	return nil
}

func (h PutPerfDataSources) createGitLabDataSource(c *codebaseApi.Codebase, dataSourceType string) error {
	config, err := h.getGitLabDataSourceConfig(c)
	if err != nil {
		return err
	}

	ds := &perfAPi.PerfDataSourceGitLab{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "PerfDataSourceGitLab",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      getDataSourceName(c.Name, dataSourceType),
			Namespace: c.Namespace,
		},
		Spec: perfAPi.PerfDataSourceGitLabSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         *config,
			PerfServerName: c.Spec.Perf.Name,
			CodebaseName:   c.Name,
		},
	}

	if err := h.client.Create(context.TODO(), ds); err != nil {
		return errors.Wrapf(err, "couldn't create PERF GitLab data source %v-%v", c.Name, dataSourceType)
	}
	return nil
}

func (h PutPerfDataSources) getJenkinsDataSourceConfig(branch, codebase, namespace string) (*perfAPi.DataSourceJenkinsConfig, error) {
	c, err := util.GetEdpComponent(h.client, jenkinsEdpComponentName, namespace)
	if err != nil {
		return nil, err
	}

	return &perfAPi.DataSourceJenkinsConfig{
		JobNames: []string{fmt.Sprintf("/%v/%v-Build-%v", codebase, strings.ToUpper(branch), codebase)},
		Url:      c.Spec.Url,
	}, nil
}

func (h PutPerfDataSources) getSonarDataSourceConfig(branch, codebase, namespace string) (*perfAPi.DataSourceSonarConfig, error) {
	c, err := util.GetEdpComponent(h.client, sonarEdpComponentName, namespace)
	if err != nil {
		return nil, err
	}

	return &perfAPi.DataSourceSonarConfig{
		ProjectKeys: []string{fmt.Sprintf("%v", codebase)},
		Url:         c.Spec.Url,
	}, nil
}

func (h PutPerfDataSources) getGitLabDataSourceConfig(codebase *codebaseApi.Codebase) (*perfAPi.DataSourceGitLabConfig, error) {
	gs, err := util.GetGitServer(h.client, codebase.Spec.GitServer, codebase.Namespace)
	if err != nil {
		return nil, err
	}

	return &perfAPi.DataSourceGitLabConfig{
		Repositories: []string{(*codebase.Spec.GitUrlPath)[1:]},
		Branches:     []string{codebase.Spec.DefaultBranch},
		Url:          modifyGitLink(gs.GitHost),
	}, nil
}

func modifyGitLink(host string) string {
	if regexp.MustCompile(`^(https:\/\/)|^(http:\/\/)`).MatchString(host) {
		return host
	}
	return fmt.Sprintf("https://%v", host)
}

func getDataSourceName(codebase, dataSourceType string) string {
	return strings.ToLower(fmt.Sprintf("%v-%v", codebase, dataSourceType))
}
