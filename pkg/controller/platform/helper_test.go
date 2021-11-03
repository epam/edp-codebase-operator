package platform

import (
	"os"
	"testing"
)

func TestIsK8S_True(t *testing.T) {
	os.Setenv("PLATFORM_TYPE", "kubernetes")
	if !IsK8S() {
		t.Error("Should fail")
	}
}

func TestIsK8S_Flase(t *testing.T) {
	os.Clearenv()
	if IsK8S() {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_K8s(t *testing.T) {
	os.Setenv("PLATFORM_TYPE", "kubernetes")
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
