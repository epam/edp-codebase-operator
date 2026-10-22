package util

import (
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AssetsDirEnv = "ASSETS_DIR"
const WorkDirEnv = "WORKING_DIR"

func GetStringP(val string) *string {
	return &val
}

// GetWorkDir returns the clone directory of a codebase under WORKING_DIR, or under the OS temp dir when WORKING_DIR
// is unset or empty. The operator image keeps /tmp writable for any UID, including the arbitrary UID OpenShift assigns.
func GetWorkDir(codebaseName, namespace string) string {
	base := os.Getenv(WorkDirEnv)
	if base == "" {
		base = os.TempDir()
	}

	return filepath.Join(base, "codebase-operator", "edp", namespace, codebaseName, "templates", codebaseName)
}

func GetAssetsDir() (string, error) {
	value, ok := os.LookupEnv(AssetsDirEnv)
	if !ok {
		return "", fmt.Errorf("ASSETS_DIR env variable is not set")
	}

	return value, nil
}

func GetOwnerReference(ownerKind string, ors []metav1.OwnerReference) (*metav1.OwnerReference, error) {
	log.V(2).Info("finding owner", "kind", ownerKind)

	if len(ors) == 0 {
		return nil, fmt.Errorf("resource doesn't have %v owner reference", ownerKind)
	}

	for _, o := range ors {
		if o.Kind == ownerKind {
			return &o, nil
		}
	}

	return nil, fmt.Errorf("resource doesn't have %v owner reference", ownerKind)
}
