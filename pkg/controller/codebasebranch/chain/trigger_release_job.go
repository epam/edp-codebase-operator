package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jfv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
)

type TriggerReleaseJob struct {
	TriggerJob
}

func (h TriggerReleaseJob) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	return h.trigger(cb, h.TriggerJob.service.TriggerReleaseJob)
}

func hasNewVersion(b *v1alpha1.CodebaseBranch) bool {
	return !util.SearchVersion(b.Status.VersionHistory, *b.Spec.Version)
}

func isJenkinsFolderAvailable(jf *jfv1alpha1.JenkinsFolder) bool {
	return jf == nil || !jf.Status.Available
}
