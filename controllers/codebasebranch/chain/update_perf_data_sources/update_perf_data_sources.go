package update_perf_data_sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	perfApi "github.com/epam/edp-perf-operator/v2/api/v1"
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

var log = ctrl.Log.WithName("update-perf-data-source-chain")

func (h UpdatePerfDataSources) ServeRequest(cb *codebaseApi.CodebaseBranch) error {
	rLog := log.WithValues("codebase", cb.Spec.CodebaseName, "branch", cb.Name)
	rLog.Info("start updating PERF data source cr...")

	if err := h.setIntermediateSuccessFields(cb, codebaseApi.PerfDataSourceCrUpdate); err != nil {
		return err
	}

	if err := h.tryToUpdateDataSourceCr(cb); err != nil {
		setFailedFields(cb, codebaseApi.PerfDataSourceCrUpdate, err.Error())
		return errors.Wrap(err, "couldn't update PerfDataSource CR")
	}

	rLog.Info("data source has been updated")

	err := handler.NextServeOrNil(h.Next, cb)
	if err != nil {
		return fmt.Errorf("failed to process next handler in chain: %w", err)
	}

	return nil
}

func (h UpdatePerfDataSources) setIntermediateSuccessFields(cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
	ctx := context.Background()
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:              model.StatusInit,
		LastTimeUpdated:     metaV1.Now(),
		Action:              action,
		Result:              codebaseApi.Success,
		Username:            "system",
		Value:               "inactive",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
	}

	err := h.Client.Status().Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch status field %q: %w", cb.Name, err)
	}

	err = h.Client.Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch resource %q: %w", cb.Name, err)
	}

	return nil
}

func setFailedFields(cb *codebaseApi.CodebaseBranch, a codebaseApi.ActionType, message string) {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:              util.StatusFailed,
		LastTimeUpdated:     metaV1.Now(),
		Username:            "system",
		Action:              a,
		Result:              codebaseApi.Error,
		DetailedMessage:     message,
		Value:               "failed",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
	}
}

func (h UpdatePerfDataSources) tryToUpdateDataSourceCr(cb *codebaseApi.CodebaseBranch) error {
	owr, err := util.GetOwnerReference(codebaseKind, cb.GetOwnerReferences())
	if err != nil {
		return errors.New("unable to get owner reference")
	}

	c, err := util.GetCodebase(h.Client, owr.Name, cb.Namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch codebase %q: %w", owr.Name, err)
	}

	if c.Spec.Perf == nil {
		log.Info("PERF server wasn't selected. skip creating PERF data source cr...",
			"codebase_name", c.Name)
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

func (h UpdatePerfDataSources) getCreateFactory() map[string]func(cb *codebaseApi.CodebaseBranch, dataSourceType string) error {
	return map[string]func(cb *codebaseApi.CodebaseBranch, dataSourceType string) error{
		jenkinsDataSourceType: h.tryToUpdateJenkinsDataSource,
		gitLabDataSourceType:  h.tryToUpdateGitLabDataSource,
	}
}

func (h UpdatePerfDataSources) tryToUpdateJenkinsDataSource(cb *codebaseApi.CodebaseBranch, dataSourceType string) error {
	ctx := context.Background()
	dsName := fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(dataSourceType))

	ds, err := h.getPerfDataSourceJenkinsCr(dsName, cb.Namespace)
	if err != nil {
		return err
	}

	jn := fmt.Sprintf("/%v/%v-Build-%v", cb.Spec.CodebaseName, strings.ToUpper(cb.Spec.BranchName), cb.Spec.CodebaseName)
	if util.ContainsString(ds.Spec.Config.JobNames, jn) {
		log.Info("perf data source already contains job", "job", jn)
		return nil
	}

	ds.Spec.Config.JobNames = append(ds.Spec.Config.JobNames, jn)

	err = h.Client.Update(ctx, ds)
	if err != nil {
		return fmt.Errorf("failed to update GitLab PerfDataSource %q: %w", dsName, err)
	}

	return nil
}

func (h UpdatePerfDataSources) tryToUpdateGitLabDataSource(cb *codebaseApi.CodebaseBranch, dataSourceType string) error {
	ctx := context.Background()
	dsName := fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(dataSourceType))

	ds, err := h.getPerfDataSourceGitLabCr(dsName, cb.Namespace)
	if err != nil {
		return err
	}

	if util.ContainsString(ds.Spec.Config.Branches, cb.Spec.BranchName) {
		log.Info("perf data source already contains branch", "branch", cb.Spec.BranchName)
		return nil
	}

	ds.Spec.Config.Branches = append(ds.Spec.Config.Branches, cb.Spec.BranchName)

	err = h.Client.Update(ctx, ds)
	if err != nil {
		return fmt.Errorf("failed to update GitLab PerfDataSource %q: %w", dsName, err)
	}

	return nil
}

func (h UpdatePerfDataSources) getPerfDataSourceJenkinsCr(name, namespace string) (*perfApi.PerfDataSourceJenkins, error) {
	ctx := context.Background()
	instance := &perfApi.PerfDataSourceJenkins{}

	err := h.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Jenkins PerfDataSource %q: %w", name, err)
	}

	return instance, nil
}

func (h UpdatePerfDataSources) getPerfDataSourceGitLabCr(name, namespace string) (*perfApi.PerfDataSourceGitLab, error) {
	ctx := context.Background()
	instance := &perfApi.PerfDataSourceGitLab{}

	err := h.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitLab PerfDataSource %q: %w", name, err)
	}

	return instance, nil
}
