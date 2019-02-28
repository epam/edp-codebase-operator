package models

type AppSettings struct {
	WorkDir               string         `json:"work_dir"`
	RepositoryUrl         string         `json:"repository_url"`
	GerritKeyPath         string         `json:"gerrit_key_path"`
	BasicPatternUrl       string         `json:"basic_pattern_url"`
	CicdNamespace         string         `json:"cicd_namespace"`
	ProjectVcsHostname    string         `json:"project_vcs_hostname"`
	ProjectVcsGroupPath   string         `json:"project_vcs_group_path"`
	ProjectVcsHostnameUrl string         `json:"project_vcs_hostname_url"`
	VcsProjectPath        string         `json:"vcs_project_path"`
	JenkinsToken          string         `json:"jenkins_token"`
	JenkinsUsername       string         `json:"jenkins_username"`
	UserSettings          UserSettings   `json:"user_settings"`
	GerritSettings        GerritSettings `json:"gerrit_settings"`
	VcsKeyPath            string         `json:"vcs_key_path"`
	VcsAutouserSshKey     string         `json:"vcs_autouser_ssh_key"`
	VcsAutouserEmail      string         `json:"vcs_autouser_email"`
	GerritPrivateKey      string         `json:"gerrit_private_key"`
	GerritPublicKey       string         `json:"gerrit_public_key"`
}

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

type VCSTool string

type GerritSettings struct {
	Config            string `json:"config"`
	ReplicationConfig string `json:"replication_config"`
	SshPort           string `json:"ssh_port"`
}

type Strategy string

const (
	Create    Strategy = "create"
	Clone     Strategy = "clone"
	BitBucket VCSTool  = "bitbucket"
	GitLab    VCSTool  = "gitlab"
)

const (
	StatusInit       = "initialized"
	StatusInProgress = "in progress"
	StatusFailed     = "failed"
	StatusReopened   = "reopened"
	StatusFinished   = "created"
)
