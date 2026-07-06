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

func TestIsEnvoyGateway_True(t *testing.T) {
	t.Setenv(GatewayTypeEnv, GatewayTypeEnvoy)

	if !IsEnvoyGateway() {
		t.Error("Should be envoy gateway")
	}
}

func TestIsEnvoyGateway_False(t *testing.T) {
	if IsEnvoyGateway() {
		t.Error("Should not be envoy gateway by default")
	}
}

func TestIsIngressGateway_True(t *testing.T) {
	if !IsIngressGateway() {
		t.Error("Should be ingress gateway by default")
	}
}

func TestIsIngressGateway_False(t *testing.T) {
	t.Setenv(GatewayTypeEnv, GatewayTypeEnvoy)

	if IsIngressGateway() {
		t.Error("Should not be ingress gateway when envoy is set")
	}
}
