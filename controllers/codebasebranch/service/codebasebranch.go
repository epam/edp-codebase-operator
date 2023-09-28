package service

import (
	"context"
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

var log = ctrl.Log.WithName("codebase_branch_service")

type CodebaseBranchService interface {
	AppendVersionToTheHistorySlice(*codebaseApi.CodebaseBranch) error
	ResetBranchBuildCounter(*codebaseApi.CodebaseBranch) error
	ResetBranchSuccessBuildCounter(*codebaseApi.CodebaseBranch) error
	updateStatus(*codebaseApi.CodebaseBranch) error
}

type CodebaseBranchServiceProvider struct {
	Client client.Client
}

var ErrJobFailed = errors.New("deletion job failed")

func (s *CodebaseBranchServiceProvider) AppendVersionToTheHistorySlice(b *codebaseApi.CodebaseBranch) error {
	if b.Spec.Version == nil {
		return nil
	}

	v := b.Spec.Version
	b.Status.VersionHistory = append(b.Status.VersionHistory, *v)

	return s.updateStatus(b)
}

func (s *CodebaseBranchServiceProvider) ResetBranchBuildCounter(cb *codebaseApi.CodebaseBranch) error {
	if cb.Status.Build == nil {
		return nil
	}

	cb.Status.Build = util.GetStringP("0")

	return s.updateStatus(cb)
}

func (s *CodebaseBranchServiceProvider) ResetBranchSuccessBuildCounter(cb *codebaseApi.CodebaseBranch) error {
	if cb.Status.LastSuccessfulBuild == nil {
		return nil
	}

	cb.Status.LastSuccessfulBuild = nil

	return s.updateStatus(cb)
}

func (s *CodebaseBranchServiceProvider) updateStatus(cb *codebaseApi.CodebaseBranch) error {
	if err := s.Client.Status().Update(context.TODO(), cb); err != nil {
		if err = s.Client.Update(context.TODO(), cb); err != nil {
			return fmt.Errorf("failed to update codebase branch status: %w", err)
		}
	}

	log.V(2).Info("codebase branch status has been updated", "name", cb.Name)

	return nil
}
