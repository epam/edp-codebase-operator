package service

import (
	"context"
	"fmt"
	"time"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/jenkins"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("codebase_branch_service")

const jenkinsJobSuccessStatus = "blue"

type CodebaseBranchService struct {
	Client client.Client
}

type JobFailedError string

func (j JobFailedError) Error() string {
	return string(j)
}

func (s *CodebaseBranchService) TriggerDeletionJob(cb *v1alpha1.CodebaseBranch) error {
	rLog := log.WithValues("codebase branch", cb.Name, "codebase", cb.Spec.CodebaseName)
	rLog.V(2).Info("start triggering deletion job")

	jc, err := initJenkinsClient(s.Client, cb.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't create jenkins client")
	}

	if err = jc.TriggerDeletionJob(cb.Spec.BranchName, cb.Spec.CodebaseName); err != nil {
		switch err.(type) {
		case jenkins.JobNotFoundError:
			rLog.Info("deletion job not found")
			return nil
		default:
			return errors.Wrap(err, "unable to trigger deletion job")
		}
	}

	rLog.Info("Deletion job has been triggered")

	rj := fmt.Sprintf("%v/job/Delete-release-%v", cb.Spec.CodebaseName, cb.Spec.CodebaseName)
	js, err := jc.GetJobStatus(rj, 10*time.Second, 50)
	if err != nil {
		return errors.Wrap(err, "unable to get deletion job status")
	}

	if js != jenkinsJobSuccessStatus {
		rLog.Info("failed to delete release", "deletion release job status", js)
		return JobFailedError("deletion job failed")
	}

	rLog.Info("release has been deleted. Status: %v", model.StatusFinished)
	return nil
}

func (s *CodebaseBranchService) TriggerReleaseJob(cb *v1alpha1.CodebaseBranch) error {
	if cb.Status.Status != model.StatusInit {
		log.Info("Release for codebase is not in init status. Skipped.",
			"release", cb.Spec.BranchName, "codebase", cb.Spec.CodebaseName)
		return nil
	}

	rLog := log.WithValues("codebase branch", cb.Name, "codebase", cb.Spec.CodebaseName)
	rLog.V(2).Info("start triggering release job")

	jc, err := initJenkinsClient(s.Client, cb.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't create jenkins client")
	}
	rLog.V(2).Info("start creating release for codebase")

	if err = jc.TriggerReleaseJob(cb.Spec.BranchName, cb.Spec.FromCommit, cb.Spec.CodebaseName); err != nil {
		return err
	}
	rLog.Info("Release job has been triggered")

	rj := fmt.Sprintf("%v/job/Create-release-%v", cb.Spec.CodebaseName, cb.Spec.CodebaseName)
	js, err := jc.GetJobStatus(rj, 10*time.Second, 50)
	if err != nil {
		return err
	}

	if js != jenkinsJobSuccessStatus {
		rLog.Info("failed to create release", "release job status", js)
		return nil
	}
	rLog.Info("release has been created", "status", model.StatusFinished)
	return nil
}

func initJenkinsClient(client client.Client, namespace string) (*jenkins.JenkinsClient, error) {
	j, err := jenkins.GetJenkins(client, namespace)
	if err != nil {
		return nil, err
	}
	jt, ju, err := jenkins.GetJenkinsCreds(client, *j, namespace)
	if err != nil {
		return nil, err
	}
	jurl := jenkins.GetJenkinsUrl(*j, namespace)
	jc, err := jenkins.Init(jurl, ju, jt)
	if err != nil {
		return nil, err
	}
	log.V(2).Info("jenkins client has been created", "url", jurl, "user", ju)
	return jc, nil
}

func (s *CodebaseBranchService) AppendVersionToTheHistorySlice(b *v1alpha1.CodebaseBranch) error {
	if b.Spec.Version == nil {
		return nil
	}
	v := b.Spec.Version
	b.Status.VersionHistory = append(b.Status.VersionHistory, *v)
	return s.updateStatus(b)
}

func (s *CodebaseBranchService) ResetBranchBuildCounter(cb *v1alpha1.CodebaseBranch) error {
	if cb.Status.Build == nil {
		return nil
	}
	cb.Status.Build = util.GetStringP("0")
	return s.updateStatus(cb)
}

func (s *CodebaseBranchService) ResetBranchSuccessBuildCounter(cb *v1alpha1.CodebaseBranch) error {
	if cb.Status.LastSuccessfulBuild == nil {
		return nil
	}

	cb.Status.LastSuccessfulBuild = nil
	return s.updateStatus(cb)
}

func (s *CodebaseBranchService) updateStatus(cb *v1alpha1.CodebaseBranch) error {
	if err := s.Client.Status().Update(context.TODO(), cb); err != nil {
		if err := s.Client.Update(context.TODO(), cb); err != nil {
			return errors.Wrap(err, "couldn't update codebase branch status")
		}
	}
	log.V(2).Info("codebase branch status has been updated", "name", cb.Name)
	return nil
}
