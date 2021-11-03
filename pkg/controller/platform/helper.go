package platform

import "os"

const (
	Openshift = "openshift"
	K8S       = "kubernetes"

	ptKey     = "PLATFORM_TYPE"
	defaultPt = Openshift
)

func lookup() string {
	if value, ok := os.LookupEnv(ptKey); ok {
		return value
	}
	return defaultPt
}

func IsK8S() bool {
	return lookup() == K8S
}

func GetPlatformType() string {
	return lookup()
}
