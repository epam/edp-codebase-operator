package update_perf_data_sources

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	perfApi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/perf-operator/v2/pkg/util/cluster"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

type UpdatePerfDataSources struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
}

const (
	codebaseKind = "Codebase"

	jenkinsDataSourceType = "Jenkins"
	gitLabDataSourceType  = "GitLab"
)

var log = logf.Log.WithName("update-perf-data-source-chain")

func (h UpdatePerfDataSources) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	rLog := log.WithValues("codebase", cb.Spec.CodebaseName, "branch", cb.Name)
	rLog.Info("start updating PERF data source cr...")
	if err := h.tryToUpdateDataSourceCr(cb); err != nil {
		return errors.Wrap(err, "couldn't update PerfDataSource CR")
	}
	rLog.Info("data source has been updated")
	return handler.NextServeOrNil(h.Next, cb)
}

func (h UpdatePerfDataSources) tryToUpdateDataSourceCr(cb *v1alpha1.CodebaseBranch) error {
	owr := cluster.GetOwnerReference(codebaseKind, cb.GetOwnerReferences())
	c, err := cluster.GetCodebase(h.Client, owr.Name, cb.Namespace)
	if err != nil {
		return err
	}

	if c.Spec.Perf == nil {
		log.Info("PERF server wasn't selected. skip creating PERF data source cr...",
			"codebase name", c.Name)
		return nil
	}

	createFactory := h.getCreateFactory()
	for _, name := range c.Spec.Perf.DataSources {
		f := createFactory[name]
		if f == nil {
			continue
		}

		if err := f(cb, name); err != nil {
			return err
		}
	}
	return nil
}

func (h UpdatePerfDataSources) getCreateFactory() map[string]func(cb *v1alpha1.CodebaseBranch, dataSourceType string) error {
	return map[string]func(cb *v1alpha1.CodebaseBranch, dataSourceType string) error{
		jenkinsDataSourceType: h.tryToUpdateJenkinsDataSource,
		gitLabDataSourceType:  h.tryToUpdateGitLabDataSource,
	}
}

func (h UpdatePerfDataSources) tryToUpdateJenkinsDataSource(cb *v1alpha1.CodebaseBranch, dataSourceType string) error {
	ds, err := h.getPerfDataSourceJenkinsCr(fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(dataSourceType)), cb.Namespace)
	if err != nil {
		return err
	}

	jn := fmt.Sprintf("/%v/%v-Build-%v", cb.Spec.CodebaseName, strings.ToUpper(cb.Spec.BranchName), cb.Spec.CodebaseName)
	if util.ContainsString(ds.Spec.Config.JobNames, jn) {
		log.Info("perf data source already contains job", "job", jn)
		return nil
	}

	ds.Spec.Config.JobNames = append(ds.Spec.Config.JobNames, jn)

	return h.Client.Update(context.TODO(), ds)
}

func (h UpdatePerfDataSources) tryToUpdateGitLabDataSource(cb *v1alpha1.CodebaseBranch, dataSourceType string) error {
	ds, err := h.getPerfDataSourceGitLabCr(fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(dataSourceType)), cb.Namespace)
	if err != nil {
		return err
	}

	if util.ContainsString(ds.Spec.Config.Branches, cb.Spec.BranchName) {
		log.Info("perf data source already contains branch", "branch", cb.Spec.BranchName)
		return nil
	}

	ds.Spec.Config.Branches = append(ds.Spec.Config.Branches, cb.Spec.BranchName)

	return h.Client.Update(context.TODO(), ds)
}

func (h UpdatePerfDataSources) getPerfDataSourceJenkinsCr(name, namespace string) (*perfApi.PerfDataSourceJenkins, error) {
	instance := &perfApi.PerfDataSourceJenkins{}
	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (h UpdatePerfDataSources) getPerfDataSourceGitLabCr(name, namespace string) (*perfApi.PerfDataSourceGitLab, error) {
	instance := &perfApi.PerfDataSourceGitLab{}
	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
