package util

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const kindName = "stub-kind"

func TestGetOwnerReference_ShouldFindOwner(t *testing.T) {
	refs := []v1.OwnerReference{
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
	refs := []v1.OwnerReference{
		{
			Kind: "fake-another-kind",
		},
	}
	ref, err := GetOwnerReference(kindName, refs)
	assert.Error(t, err)
	assert.Nil(t, ref)
}
