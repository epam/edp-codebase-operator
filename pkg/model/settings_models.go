package model

type CodebaseSettings struct {
	Name                  string         `json:"name"`
	Type                  string         `json:"type"`
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
	JenkinsUrl            string         `json:"jenkins_url"`
	UserSettings          UserSettings   `json:"user_settings"`
	GerritSettings        GerritSettings `json:"gerrit_settings"`
	VcsKeyPath            string         `json:"vcs_key_path"`
	VcsAutouserSshKey     string         `json:"vcs_autouser_ssh_key"`
	VcsAutouserEmail      string         `json:"vcs_autouser_email"`
	GerritPrivateKey      string         `json:"gerrit_private_key"`
	GerritPublicKey       string         `json:"gerrit_public_key"`
	VcsSshUrl             string         `json:"vcs_ssh_url"`
	GerritHost            string         `json:"gerrit_host"`
	RepositoryPath        string         `json:"repositoryPath"`
	Lang                  string         `json:"language"`
	GitServer             GitServer      `json:"gitServer"`
	Framework             string         `json:"framework"`
	JobProvisioning       string         `json:"jobProvisioning"`
	Strategy              string         `json:"strategy"`
	DeploymentScript      string         `json:"deploymentScript"`
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

const (
	BitBucket        VCSTool = "bitbucket"
	GitLab           VCSTool = "gitlab"
	StatusInit               = "initialized"
	StatusInProgress         = "in progress"
	StatusFailed             = "failed"
	StatusReopened           = "reopened"
	StatusFinished           = "created"
	JavaScript       string  = "javascript"
	Java             string  = "java"
	DotNet           string  = "dotnet"
)
