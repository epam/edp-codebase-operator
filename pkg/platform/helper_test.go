package platform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsK8S_True(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", "kubernetes")
	require.NoError(t, err)

	if !IsK8S() {
		t.Error("Should fail")
	}
}

func TestIsK8S_False(t *testing.T) {
	os.Clearenv()

	if IsK8S() {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_K8s(t *testing.T) {
	err := os.Setenv("PLATFORM_TYPE", "kubernetes")
	require.NoError(t, err)

	if GetPlatformType() == Openshift {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_Openshift(t *testing.T) {
	os.Clearenv()

	if GetPlatformType() == K8S {
		t.Error("Should fail")
	}
}
