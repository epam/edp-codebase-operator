package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func Test_hasNewVersion(t *testing.T) {
	type args struct {
		b *codebaseApi.CodebaseBranch
	}

	version1 := "0.0.0-SNAPSHOT"
	version2 := "1.0.0-SNAPSHOT"
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
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
			}, false, assert.NoError},
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
			}, true, assert.NoError},
		{"Codebasebranch has no version",
			args{
				b: &codebaseApi.CodebaseBranch{},
			}, false, assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasNewVersion(tt.args.b)
			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}
