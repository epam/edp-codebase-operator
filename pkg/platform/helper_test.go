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

func TestIsK8S_DefaultTrue(t *testing.T) {
	if !IsK8S() {
		t.Error("kubernetes should be the default platform when PLATFORM_TYPE is unset")
	}
}

func TestIsOpenshift_True(t *testing.T) {
	t.Setenv("PLATFORM_TYPE", "openshift")

	if !IsOpenshift() {
		t.Error("Should fail")
	}
}

func TestIsOpenshift_DefaultFalse(t *testing.T) {
	if IsOpenshift() {
		t.Error("openshift should not be the default platform when PLATFORM_TYPE is unset")
	}
}

func TestGetPlatformType_K8s(t *testing.T) {
	t.Setenv("PLATFORM_TYPE", "kubernetes")

	if GetPlatformType() == Openshift {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_Openshift(t *testing.T) {
	t.Setenv("PLATFORM_TYPE", "openshift")

	if GetPlatformType() == K8S {
		t.Error("Should fail")
	}
}

func TestGetPlatformType_DefaultKubernetes(t *testing.T) {
	if GetPlatformType() != K8S {
		t.Error("kubernetes should be the default platform when PLATFORM_TYPE is unset")
	}
}

func TestIsEnvoy_True(t *testing.T) {
	t.Setenv(IngressControllerEnv, IngressControllerEnvoy)

	if !IsEnvoy() {
		t.Error("Should fail")
	}
}

func TestIsEnvoy_DefaultFalse(t *testing.T) {
	if IsEnvoy() {
		t.Error("envoy should not be the default ingress controller when INGRESS_CONTROLLER_TYPE is unset")
	}
}

func TestGatewayName(t *testing.T) {
	t.Setenv(GatewayNameEnv, "main-gateway")

	if GatewayName() != "main-gateway" {
		t.Error("Should fail")
	}
}

func TestGatewayName_DefaultEmpty(t *testing.T) {
	if GatewayName() != "" {
		t.Error("Should fail")
	}
}

func TestGatewayNamespace(t *testing.T) {
	t.Setenv(GatewayNamespaceEnv, "envoy-gateway-system")

	if GatewayNamespace() != "envoy-gateway-system" {
		t.Error("Should fail")
	}
}

func TestGatewayNamespace_DefaultEmpty(t *testing.T) {
	if GatewayNamespace() != "" {
		t.Error("Should fail")
	}
}
