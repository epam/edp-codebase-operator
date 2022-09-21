package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const kindName = "stub-kind"

func TestGetOwnerReference_ShouldFindOwner(t *testing.T) {
	refs := []metaV1.OwnerReference{
		{
			Kind: kindName,
		},
	}
	ref, err := GetOwnerReference(kindName, refs)
	assert.NoError(t, err)
	assert.Equal(t, kindName, ref.Kind)
}

func TestGetOwnerReference_ShouldReturnErrorBecauseOfMissingOfPassedArg(t *testing.T) {
	ref, err := GetOwnerReference(kindName, nil)
	assert.Error(t, err)
	assert.Nil(t, ref)
}

func TestGetOwnerReference_ShouldNotFindOwner(t *testing.T) {
	refs := []metaV1.OwnerReference{
		{
			Kind: "fake-another-kind",
		},
	}
	ref, err := GetOwnerReference(kindName, refs)
	assert.Error(t, err)
	assert.Nil(t, ref)
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
		{"1", args{codebaseName: "test", namespace: "stub-namespace"}, "/home/codebase-operator/edp/stub-namespace/test/templates/test"},
		{"2", args{codebaseName: "cb-name", namespace: "stub-namespace"}, "/home/codebase-operator/edp/stub-namespace/cb-name/templates/cb-name"},
		{"3", args{codebaseName: "demo", namespace: "stub-namespace"}, "/home/codebase-operator/edp/stub-namespace/demo/templates/demo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetWorkDir(tt.args.codebaseName, tt.args.namespace); got != tt.want {
				t.Errorf("GetWorkDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkDir_WithCustomPathShouldWork(t *testing.T) {
	err := os.Setenv("WORKING_DIR", "/CUSTOM_PATH")
	require.NoError(t, err)

	type args struct {
		codebaseName string
		namespace    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"1", args{codebaseName: "test", namespace: "fake-ns"}, "/CUSTOM_PATH/codebase-operator/edp/fake-ns/test/templates/test"},
		{"2", args{codebaseName: "cb-name", namespace: "fake-ns"}, "/CUSTOM_PATH/codebase-operator/edp/fake-ns/cb-name/templates/cb-name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetWorkDir(tt.args.codebaseName, tt.args.namespace); got != tt.want {
				t.Errorf("GetWorkDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAssetsDir_ShouldPass(t *testing.T) {
	err := os.Setenv("ASSETS_DIR", "/tmp")
	require.NoError(t, err)

	ad := GetAssetsDir()
	assert.Equal(t, ad, "/tmp")
}
