package platform

import "os"

const (
	Openshift = "openshift"
	K8S       = "kubernetes"

	TypeEnv   = "PLATFORM_TYPE"
	defaultPt = K8S

	IngressControllerEnv   = "INGRESS_CONTROLLER_TYPE"
	IngressControllerNginx = "nginx"
	IngressControllerEnvoy = "envoy"

	GatewayNameEnv      = "GATEWAY_NAME"
	GatewayNamespaceEnv = "GATEWAY_NAMESPACE"
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

// IsEnvoy returns true when the Envoy Gateway (Gateway API) ingress controller is selected.
func IsEnvoy() bool {
	return os.Getenv(IngressControllerEnv) == IngressControllerEnvoy
}

// GatewayName returns the name of the parent Gateway API Gateway resource.
func GatewayName() string {
	return os.Getenv(GatewayNameEnv)
}

// GatewayNamespace returns the namespace of the parent Gateway API Gateway resource.
func GatewayNamespace() string {
	return os.Getenv(GatewayNamespaceEnv)
}
