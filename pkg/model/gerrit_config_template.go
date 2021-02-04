package model

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type ConfigGoTemplating struct {
	Lang         string          `json:"lang"`
	Route        *v1alpha1.Route `json:"route"`
	Name         string
	PlatformType string
	DnsWildcard  string
	Framework    string
}
