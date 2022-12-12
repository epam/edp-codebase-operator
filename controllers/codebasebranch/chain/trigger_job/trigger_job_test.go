package trigger_job

import (
	"testing"

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
