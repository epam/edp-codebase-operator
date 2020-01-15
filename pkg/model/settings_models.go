package model

type UserSettings struct {
	DnsWildcard            string  `json:"dns_wildcard"`
	EdpName                string  `json:"edp_name"`
	EdpVersion             string  `json:"edp_version"`
	PerfIntegrationEnabled bool    `json:"perf_integration_enabled"`
	VcsGroupNameUrl        string  `json:"vcs_group_name_url"`
	VcsIntegrationEnabled  bool    `json:"vcs_integration_enabled"`
	VcsSshPort             string  `json:"vcs_ssh_port"`
	VcsToolName            VCSTool `json:"vcs_tool_name"`
}

type EnvSettings struct {
	Name     string    `json:"name"`
	Triggers []Trigger `json:"triggers"`
}

type Trigger struct {
	Type string `json:"type"`
}

type VCSTool string

type GerritSettings struct {
	Config            string `json:"config"`
	ReplicationConfig string `json:"replication_config"`
	SshPort           int32  `json:"ssh_port"`
}

type SshConfig struct {
	CicdNamespace         string
	SshPort               int32
	GerritKeyPath         string
	VcsIntegrationEnabled bool
	ProjectVcsHostname    string
	VcsSshPort            string
	VcsKeyPath            string
}

type Vcs struct {
	VcsSshUrl             string
	VcsIntegrationEnabled bool
	VcsToolName           VCSTool
	ProjectVcsHostnameUrl string
	ProjectVcsGroupPath   string
}

const (
	BitBucket        VCSTool = "bitbucket"
	GitLab           VCSTool = "gitlab"
	StatusInit               = "initialized"
	StatusInProgress         = "in progress"
	StatusFailed             = "failed"
	StatusReopened           = "reopened"
	StatusFinished           = "created"
	JavaScript               = "javascript"
	Java                     = "java"
	DotNet                   = "dotnet"
	GroovyPipeline           = "groovy-pipeline"
)
