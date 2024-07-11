package platform

import "os"

const (
	Openshift = "openshift"
	K8S       = "kubernetes"

	TypeEnv   = "PLATFORM_TYPE"
	defaultPt = Openshift
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
