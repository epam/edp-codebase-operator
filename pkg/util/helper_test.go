package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetOwnerReference(t *testing.T) {
	t.Parallel()

	type args struct {
		ownerKind string
		ors       []metav1.OwnerReference
	}

	tests := []struct {
		name    string
		args    args
		want    *metav1.OwnerReference
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should find owner",
			args: args{
				ownerKind: "stub-kind",
				ors: []metav1.OwnerReference{
					{
						Kind: "stub-kind",
					},
				},
			},
			want: &metav1.OwnerReference{
				Kind: "stub-kind",
			},
			wantErr: require.NoError,
		},
		{
			name: "should return error because of the nil refs",
			args: args{
				ownerKind: "stub-kind",
				ors:       nil,
			},
			want:    nil,
			wantErr: require.Error,
		},
		{
			name: "should not find owner",
			args: args{
				ownerKind: "stub-kind",
				ors: []metav1.OwnerReference{
					{
						Kind: "not-a-stub-kind",
					},
				},
			},
			want:    nil,
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetOwnerReference(tt.args.ownerKind, tt.args.ors)

			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetWorkDir(t *testing.T) {
	t.Setenv("TMPDIR", "/stub-tmp")

	tempWorkDir := filepath.Join(os.TempDir(), "codebase-operator", "edp", "stub-namespace", "test", "templates", "test")

	tests := []struct {
		name    string
		workDir string
		unset   bool
		want    string
	}{
		{
			name:  "defaults to the temp dir when WORKING_DIR is unset",
			unset: true,
			want:  tempWorkDir,
		},
		{
			name: "defaults to the temp dir when WORKING_DIR is empty",
			want: tempWorkDir,
		},
		{
			name:    "uses WORKING_DIR",
			workDir: "/CUSTOM_PATH",
			want:    filepath.Join("/CUSTOM_PATH", "codebase-operator", "edp", "stub-namespace", "test", "templates", "test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(WorkDirEnv, tt.workDir)

			if tt.unset {
				require.NoError(t, os.Unsetenv(WorkDirEnv))
			}

			assert.Equal(t, tt.want, GetWorkDir("test", "stub-namespace"))
		})
	}
}

func TestGetAssetsDir_ShouldPass(t *testing.T) {
	t.Setenv(AssetsDirEnv, "/tmp")

	ad, err := GetAssetsDir()
	assert.Equal(t, ad, "/tmp")
	assert.NoError(t, err)
}
