package trigger_job

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jfv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
)

func Test_isJenkinsFolderAvailable(t *testing.T) {
	type args struct {
		jf *jfv1alpha1.JenkinsFolder
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Jenkinsfolder is available", args{jf: &jenkinsv1alpha1.JenkinsFolder{Status: jenkinsv1alpha1.JenkinsFolderStatus{Available: true}}}, true},
		{"Jenkinsfolder is NOT available", args{jf: &jenkinsv1alpha1.JenkinsFolder{Status: jenkinsv1alpha1.JenkinsFolderStatus{Available: false}}}, false},
		{"Jenkinsfolder is NOT available", args{jf: nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJenkinsFolderAvailable(tt.args.jf); got != tt.want {
				t.Errorf("isJenkinsFolderAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasNewVersion(t *testing.T) {
	type args struct {
		b *v1alpha1.CodebaseBranch
	}
	var version1 string = "0.0.0-SNAPSHOT"
	var version2 string = "1.0.0-SNAPSHOT"
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Codebasebranch Doesn't have new version",
			args{
				b: &v1alpha1.CodebaseBranch{
					Spec: v1alpha1.CodebaseBranchSpec{
						Version: &version1,
					},
					Status: v1alpha1.CodebaseBranchStatus{
						VersionHistory: []string{"0.0.0-SNAPSHOT"},
					},
				},
			}, false},
		{"Codebasebranch has new version",
			args{
				b: &v1alpha1.CodebaseBranch{
					Spec: v1alpha1.CodebaseBranchSpec{
						Version: &version2,
					},
					Status: v1alpha1.CodebaseBranchStatus{
						VersionHistory: []string{"0.0.0-SNAPSHOT", "0.0.1-SNAPSHOT"},
					},
				},
			}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNewVersion(tt.args.b); got != tt.want {
				t.Errorf("hasNewVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
