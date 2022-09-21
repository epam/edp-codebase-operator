package trigger_job

import (
	"testing"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func Test_isJenkinsFolderAvailable(t *testing.T) {
	type args struct {
		jf *jenkinsApi.JenkinsFolder
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Jenkinsfolder is available", args{jf: &jenkinsApi.JenkinsFolder{Status: jenkinsApi.JenkinsFolderStatus{Available: true}}}, true},
		{"Jenkinsfolder is NOT available", args{jf: &jenkinsApi.JenkinsFolder{Status: jenkinsApi.JenkinsFolderStatus{Available: false}}}, false},
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
		b *codebaseApi.CodebaseBranch
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
				b: &codebaseApi.CodebaseBranch{
					Spec: codebaseApi.CodebaseBranchSpec{
						Version: &version1,
					},
					Status: codebaseApi.CodebaseBranchStatus{
						VersionHistory: []string{"0.0.0-SNAPSHOT"},
					},
				},
			}, false},
		{"Codebasebranch has new version",
			args{
				b: &codebaseApi.CodebaseBranch{
					Spec: codebaseApi.CodebaseBranchSpec{
						Version: &version2,
					},
					Status: codebaseApi.CodebaseBranchStatus{
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
