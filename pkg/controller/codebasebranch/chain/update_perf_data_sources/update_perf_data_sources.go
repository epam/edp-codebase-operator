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

	for _, name := range c.Spec.Perf.DataSources {
		if name != "Jenkins" {
			continue
		}

		ds, err := h.getPerfDataSourceCr(fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(name)), cb.Namespace)
		if err != nil {
			return err
		}

		jn := fmt.Sprintf("/%v/%v-Build-%v", cb.Spec.CodebaseName, strings.ToUpper(cb.Spec.BranchName), cb.Spec.CodebaseName)
		if util.ContainsString(ds.Spec.Config.JobNames, jn) {
			log.Info("perf data source already contains job", "job", jn)
			return nil
		}

		ds.Spec.Config.JobNames = append(ds.Spec.Config.JobNames, jn)

		if err := h.Client.Update(context.TODO(), ds); err != nil {
			return err
		}
	}
	return nil
}

func (h UpdatePerfDataSources) getPerfDataSourceCr(name, namespace string) (*perfApi.PerfDataSource, error) {
	instance := &perfApi.PerfDataSource{}
	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
