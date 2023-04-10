package platform

import (
	"testing"
)

func TestIsK8S_True(t *testing.T) {
	t.Setenv("PLATFORM_TYPE", "kubernetes")

	if !IsK8S() {
		t.Error("Should fail")
	}
}

func TestIsK8S_False(t *testing.T) {
	if IsK8S() {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_K8s(t *testing.T) {
	t.Setenv("PLATFORM_TYPE", "kubernetes")

	if GetPlatformType() == Openshift {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_Openshift(t *testing.T) {
	if GetPlatformType() == K8S {
		t.Error("Should fail")
	}
}
