package update_perf_data_sources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	perfApi "github.com/epam/edp-perf-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
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

func (h UpdatePerfDataSources) ServeRequest(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx).WithName("update-perf-data-source")

	log.Info("Start updating PERF data source CR")

	if err := h.setIntermediateSuccessFields(cb, codebaseApi.PerfDataSourceCrUpdate); err != nil {
		return err
	}

	if err := h.tryToUpdateDataSourceCr(ctx, cb); err != nil {
		setFailedFields(cb, codebaseApi.PerfDataSourceCrUpdate, err.Error())
		return fmt.Errorf("failed to update PerfDataSource CR: %w", err)
	}

	log.Info("Data source has been updated")

	err := handler.NextServeOrNil(ctx, h.Next, cb)
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
		Git:                 cb.Status.Git,
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
		Git:                 cb.Status.Git,
	}
}

func (h UpdatePerfDataSources) tryToUpdateDataSourceCr(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx)

	owr, err := util.GetOwnerReference(codebaseKind, cb.GetOwnerReferences())
	if err != nil {
		return errors.New("failed to get owner reference")
	}

	c := &codebaseApi.Codebase{}
	if err = h.Client.Get(ctx, types.NamespacedName{
		Name:      owr.Name,
		Namespace: cb.Namespace,
	}, c); err != nil {
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

		if err := f(ctx, cb, name); err != nil {
			return err
		}
	}

	return nil
}

func (h UpdatePerfDataSources) getCreateFactory() map[string]func(ctx context.Context, cb *codebaseApi.CodebaseBranch, dataSourceType string) error {
	return map[string]func(ctx context.Context, cb *codebaseApi.CodebaseBranch, dataSourceType string) error{
		jenkinsDataSourceType: h.tryToUpdateJenkinsDataSource,
		gitLabDataSourceType:  h.tryToUpdateGitLabDataSource,
	}
}

func (h UpdatePerfDataSources) tryToUpdateJenkinsDataSource(ctx context.Context, cb *codebaseApi.CodebaseBranch, dataSourceType string) error {
	log := ctrl.LoggerFrom(ctx)

	dsName := fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(dataSourceType))

	ds, err := h.getPerfDataSourceJenkinsCr(dsName, cb.Namespace)
	if err != nil {
		return err
	}

	jn := fmt.Sprintf("/%v/%v-Build-%v", cb.Spec.CodebaseName, strings.ToUpper(cb.Spec.BranchName), cb.Spec.CodebaseName)
	if slices.Contains(ds.Spec.Config.JobNames, jn) {
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

func (h UpdatePerfDataSources) tryToUpdateGitLabDataSource(ctx context.Context, cb *codebaseApi.CodebaseBranch, dataSourceType string) error {
	log := ctrl.LoggerFrom(ctx)

	dsName := fmt.Sprintf("%v-%v", cb.Spec.CodebaseName, strings.ToLower(dataSourceType))

	ds, err := h.getPerfDataSourceGitLabCr(dsName, cb.Namespace)
	if err != nil {
		return err
	}

	if slices.Contains(ds.Spec.Config.Branches, cb.Spec.BranchName) {
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
