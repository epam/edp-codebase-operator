package model

type UserSettings struct {
	DnsWildcard string `json:"dns_wildcard"`
}

type EnvSettings struct {
	Name     string    `json:"name"`
	Triggers []Trigger `json:"triggers"`
}

type Trigger struct {
	Type string `json:"type"`
}

type VCSTool string

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
	VcsUsername           string
	VcsPassword           string
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
