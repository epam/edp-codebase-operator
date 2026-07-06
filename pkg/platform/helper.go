package platform

import "os"

const (
	Openshift = "openshift"
	K8S       = "kubernetes"

	TypeEnv   = "PLATFORM_TYPE"
	defaultPt = Openshift

	GatewayTypeEnv    = "GATEWAY_TYPE"
	GatewayTypeIngress = "ingress"
	GatewayTypeEnvoy   = "envoy"

	defaultGateway = GatewayTypeIngress

	GatewayNameEnv       = "GATEWAY_NAME"
	GatewayNamespaceEnv  = "GATEWAY_NAMESPACE"
	defaultGatewayName   = "main-gateway"
	defaultGatewayNamespace = "envoy-gateway-system"
)

func lookup() string {
	if value, ok := os.LookupEnv(TypeEnv); ok {
		return value
	}

	return defaultPt
}

func IsK8S() bool {
	return lookup() == K8S
}

func IsOpenshift() bool {
	return lookup() == Openshift
}

func GetPlatformType() string {
	return lookup()
}

func lookupGateway() string {
	if value, ok := os.LookupEnv(GatewayTypeEnv); ok {
		return value
	}

	return defaultGateway
}

func IsEnvoyGateway() bool {
	return lookupGateway() == GatewayTypeEnvoy
}

func IsIngressGateway() bool {
	return lookupGateway() == GatewayTypeIngress
}

func GetGatewayName() string {
	if v, ok := os.LookupEnv(GatewayNameEnv); ok && v != "" {
		return v
	}

	return defaultGatewayName
}

func GetGatewayNamespace() string {
	if v, ok := os.LookupEnv(GatewayNamespaceEnv); ok {
		return v
	}

	return defaultGatewayNamespace
}
