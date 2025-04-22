package util

import (
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
	type args struct {
		codebaseName string
		namespace    string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{codebaseName: "test", namespace: "stub-namespace"},
			want: "/home/codebase-operator/edp/stub-namespace/test/templates/test",
		},
		{
			name: "2",
			args: args{codebaseName: "cb-name", namespace: "stub-namespace"},
			want: "/home/codebase-operator/edp/stub-namespace/cb-name/templates/cb-name",
		},
		{
			name: "3",
			args: args{codebaseName: "demo", namespace: "stub-namespace"},
			want: "/home/codebase-operator/edp/stub-namespace/demo/templates/demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWorkDir(tt.args.codebaseName, tt.args.namespace)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetWorkDir_WithCustomPathShouldWork(t *testing.T) {
	t.Setenv("WORKING_DIR", "/CUSTOM_PATH")

	type args struct {
		codebaseName string
		namespace    string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{codebaseName: "test", namespace: "fake-ns"},
			want: "/CUSTOM_PATH/codebase-operator/edp/fake-ns/test/templates/test",
		},
		{
			name: "2",
			args: args{codebaseName: "cb-name", namespace: "fake-ns"},
			want: "/CUSTOM_PATH/codebase-operator/edp/fake-ns/cb-name/templates/cb-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWorkDir(tt.args.codebaseName, tt.args.namespace)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAssetsDir_ShouldPass(t *testing.T) {
	t.Setenv(AssetsDirEnv, "/tmp")

	ad, err := GetAssetsDir()
	assert.Equal(t, ad, "/tmp")
	assert.NoError(t, err)
}
