package codebase

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// BenchmarkCodebaseBranchListDeepCopy measures what a cached List costs: the informer
// cache deep-copies every object it returns. BranchCodebaseNameIndex shrinks that set to
// the branches of one codebase, so this is the per-object price it saves.
//
// The result is a lower bound, as cached objects also carry managedFields.
func BenchmarkCodebaseBranchListDeepCopy(b *testing.B) {
	for _, branches := range []int{10, 500, 5000} {
		b.Run(fmt.Sprintf("branches=%d", branches), func(b *testing.B) {
			list := benchBranchList(branches)

			b.ReportAllocs()

			for b.Loop() {
				_ = list.DeepCopy()
			}
		})
	}
}

func benchBranchList(size int) *codebaseApi.CodebaseBranchList {
	list := &codebaseApi.CodebaseBranchList{Items: make([]codebaseApi.CodebaseBranch, 0, size)}

	for i := range size {
		list.Items = append(list.Items, codebaseApi.CodebaseBranch{
			ObjectMeta: metav1.ObjectMeta{
				Name:            fmt.Sprintf("app-%d-main", i),
				Namespace:       "krci",
				ResourceVersion: "123456",
				Labels:          map[string]string{"app.edp.epam.com/codebaseName": fmt.Sprintf("app-%d", i)},
			},
			Spec: codebaseApi.CodebaseBranchSpec{
				CodebaseName: fmt.Sprintf("app-%d", i),
				BranchName:   "main",
				Pipelines:    map[string]string{"review": "review-pipeline", "build": "build-pipeline"},
			},
			Status: codebaseApi.CodebaseBranchStatus{
				LastTimeUpdated: metav1.Now(),
				Status:          "created",
				Value:           "active",
				Action:          codebaseApi.ActionType("codebase_branch_provisioning"),
				Result:          codebaseApi.Success,
				Username:        "system",
				VersionHistory:  []string{"0.1.0-SNAPSHOT", "0.2.0-SNAPSHOT"},
				Conditions: []metav1.Condition{{
					Type:               "Stale",
					Status:             metav1.ConditionFalse,
					Reason:             "ExistsInGit",
					Message:            "Branch exists in the git repository",
					LastTransitionTime: metav1.NewTime(time.Unix(0, 0)),
				}},
			},
		})
	}

	return list
}
