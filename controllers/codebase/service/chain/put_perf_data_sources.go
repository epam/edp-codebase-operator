package chain

import (
	"context"
	"fmt"
	"strings"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	perfAPi "github.com/epam/edp-perf-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutPerfDataSources struct {
	client client.Client
}

const (
	jenkinsEdpComponentName = "jenkins"
	sonarEdpComponentName   = "sonar"

	jenkinsDataSourceType = "Jenkins"
	sonarDataSourceType   = "Sonar"
	gitLabDataSourceType  = "GitLab"
)

func NewPutPerfDataSources(c client.Client) *PutPerfDataSources {
	return &PutPerfDataSources{client: c}
}

func (h *PutPerfDataSources) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating PERF data source")

	if err := h.tryToCreateDataSourceCr(ctx, c); err != nil {
		return fmt.Errorf("failed to create PerfDataSource CR: %w", err)
	}

	log.Info("PERF data source has been created")

	return nil
}

func (h *PutPerfDataSources) tryToCreateDataSourceCr(ctx context.Context, c *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if c.Spec.Perf == nil {
		log.Info("PERF server wasn't selected. skip creating PERF data source")
		return nil
	}

	createFactory := h.getCreateFactory()
	for _, name := range c.Spec.Perf.DataSources {
		if err := createFactory[name](ctx, c, name); err != nil {
			return err
		}
	}

	return nil
}

func (h *PutPerfDataSources) getCreateFactory() map[string]func(context.Context, *codebaseApi.Codebase, string) error {
	return map[string]func(context.Context, *codebaseApi.Codebase, string) error{
		jenkinsDataSourceType: h.tryToCreateJenkinsDataSource,
		sonarDataSourceType:   h.tryToCreateSonarDataSource,
		gitLabDataSourceType:  h.tryToCreateGitLabDataSource,
	}
}

func (h *PutPerfDataSources) tryToCreateJenkinsDataSource(ctx context.Context, c *codebaseApi.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSourceJenkins{}
	name := getDataSourceName(c.Name, dataSourceType)

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return h.createJenkinsDataSource(ctx, c, dataSourceType)
		}

		return fmt.Errorf("failed to fetch 'PerfDataSourceJenkins' resource %q: %w", name, err)
	}

	return nil
}

func (h *PutPerfDataSources) tryToCreateSonarDataSource(ctx context.Context, c *codebaseApi.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSourceSonar{}
	name := getDataSourceName(c.Name, dataSourceType)

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return h.createSonarDataSource(ctx, c, dataSourceType)
		}

		return fmt.Errorf("failed to fetch 'PerfDataSourceSonar' resource %q: %w", name, err)
	}

	return nil
}

func (h *PutPerfDataSources) tryToCreateGitLabDataSource(ctx context.Context, c *codebaseApi.Codebase, dataSourceType string) error {
	ds := &perfAPi.PerfDataSourceGitLab{}
	name := getDataSourceName(c.Name, dataSourceType)

	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace,
	}, ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return h.createGitLabDataSource(ctx, c, dataSourceType)
		}

		return fmt.Errorf("failed to fetch 'PerfDataSourceGitLab' resource %q: %w", name, err)
	}

	return nil
}

func (h *PutPerfDataSources) createJenkinsDataSource(ctx context.Context, c *codebaseApi.Codebase, dataSourceType string) error {
	config, err := h.getJenkinsDataSourceConfig(c.Spec.DefaultBranch, c.Name, c.Namespace)
	if err != nil {
		return err
	}

	typeMeta, objMeta := makeK8sDataSourceMeta(c, dataSourceType)
	ds := &perfAPi.PerfDataSourceJenkins{
		TypeMeta:   typeMeta,
		ObjectMeta: objMeta,
		Spec: perfAPi.PerfDataSourceJenkinsSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         *config,
			PerfServerName: c.Spec.Perf.Name,
			CodebaseName:   c.Name,
		},
	}

	err = h.client.Create(ctx, ds)
	if err != nil {
		return fmt.Errorf("failed to create PERF Jenkins data source %v-%v: %w", c.Name, dataSourceType, err)
	}

	return nil
}

func (h *PutPerfDataSources) createSonarDataSource(ctx context.Context, c *codebaseApi.Codebase, dataSourceType string) error {
	config, err := h.getSonarDataSourceConfig(c.Name, c.Namespace)
	if err != nil {
		return err
	}

	typeMeta, objMeta := makeK8sDataSourceMeta(c, dataSourceType)
	ds := &perfAPi.PerfDataSourceSonar{
		TypeMeta:   typeMeta,
		ObjectMeta: objMeta,
		Spec: perfAPi.PerfDataSourceSonarSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         *config,
			PerfServerName: c.Spec.Perf.Name,
			CodebaseName:   c.Name,
		},
	}

	err = h.client.Create(ctx, ds)
	if err != nil {
		return fmt.Errorf("failed to create PERF Sonar data source %v-%v: %w", c.Name, dataSourceType, err)
	}

	return nil
}

func (h *PutPerfDataSources) createGitLabDataSource(ctx context.Context, c *codebaseApi.Codebase, dataSourceType string) error {
	config, err := h.getGitLabDataSourceConfig(c)
	if err != nil {
		return err
	}

	typeMeta, objMeta := makeK8sDataSourceMeta(c, dataSourceType)
	ds := &perfAPi.PerfDataSourceGitLab{
		TypeMeta:   typeMeta,
		ObjectMeta: objMeta,
		Spec: perfAPi.PerfDataSourceGitLabSpec{
			Name:           dataSourceType,
			Type:           strings.ToUpper(dataSourceType),
			Config:         *config,
			PerfServerName: c.Spec.Perf.Name,
			CodebaseName:   c.Name,
		},
	}

	err = h.client.Create(ctx, ds)
	if err != nil {
		return fmt.Errorf("failed to create PERF GitLab data source %v-%v: %w", c.Name, dataSourceType, err)
	}

	return nil
}

func (h *PutPerfDataSources) getJenkinsDataSourceConfig(branch, codebase, namespace string) (*perfAPi.DataSourceJenkinsConfig, error) {
	c, err := util.GetEdpComponent(h.client, jenkinsEdpComponentName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch EDP Component: %w", err)
	}

	return &perfAPi.DataSourceJenkinsConfig{
		JobNames: []string{fmt.Sprintf("/%v/%v-Build-%v", codebase, strings.ToUpper(branch), codebase)},
		Url:      c.Spec.Url,
	}, nil
}

func (h *PutPerfDataSources) getSonarDataSourceConfig(codebase, namespace string) (*perfAPi.DataSourceSonarConfig, error) {
	c, err := util.GetEdpComponent(h.client, sonarEdpComponentName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch EDP Component: %w", err)
	}

	return &perfAPi.DataSourceSonarConfig{
		ProjectKeys: []string{codebase},
		Url:         c.Spec.Url,
	}, nil
}

func (h *PutPerfDataSources) getGitLabDataSourceConfig(codebase *codebaseApi.Codebase) (*perfAPi.DataSourceGitLabConfig, error) {
	gs, err := util.GetGitServer(h.client, codebase.Spec.GitServer, codebase.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Git Server: %w", err)
	}

	return &perfAPi.DataSourceGitLabConfig{
		Repositories: []string{(*codebase.Spec.GitUrlPath)[1:]},
		Branches:     []string{codebase.Spec.DefaultBranch},
		Url:          util.GetHostWithProtocol(gs.GitHost),
	}, nil
}

func makeK8sDataSourceMeta(c *codebaseApi.Codebase, dataSourceType string) (metaV1.TypeMeta, metaV1.ObjectMeta) {
	const apiVersion = "v2.edp.epam.com/v1"

	objMeta := metaV1.ObjectMeta{
		Name:      getDataSourceName(c.Name, dataSourceType),
		Namespace: c.Namespace,
	}

	switch dataSourceType {
	case jenkinsDataSourceType:
		return metaV1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "PerfDataSourceJenkins",
		}, objMeta
	case sonarDataSourceType:
		return metaV1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "PerfDataSourceSonar",
		}, objMeta
	case gitLabDataSourceType:
		return metaV1.TypeMeta{
				APIVersion: apiVersion,
				Kind:       "PerfDataSourceGitLab",
			},
			objMeta
	}

	return metaV1.TypeMeta{}, metaV1.ObjectMeta{}
}

func getDataSourceName(codebaseName, dataSourceType string) string {
	return strings.ToLower(fmt.Sprintf("%v-%v", codebaseName, dataSourceType))
}
