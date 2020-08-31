package util

import "fmt"

func GetStringP(val string) *string {
	return &val
}

func GetWorkDir(codebaseName, namespace string) string {
	return fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v/%v", namespace, codebaseName, "templates", codebaseName)
}
