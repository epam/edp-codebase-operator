package model

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type GerritConfigGoTemplating struct {
	Lang        string             `json:"lang"`
	Route       *v1alpha1.Route    `json:"route"`
	Database    *v1alpha1.Database `json:"database"`
	Name        string
	DnsWildcard string
}
