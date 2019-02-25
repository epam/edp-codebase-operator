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
	PerfSettings          PerfSettings   `json:"perf_settings"`
	GerritSettings        GerritSettings `json:"gerrit_settings"`
	VcsKeyPath            string         `json:"vcs_key_path"`
	VcsAutouserSshKey     string         `json:"vcs_autouser_ssh_key"`
	VcsAutouserEmail      string         `json:"vcs_autouser_email"`
	GerritPrivateKey      string         `json:"gerrit_private_key"`
	GerritPublicKey       string         `json:"gerrit_public_key"`
}

type UserSettings struct {
	DnsWildcard            string `json:"dns_wildcard"`
	EdpName                string `json:"edp_name"`
	EdpVersion             string `json:"edp_version"`
	PerfIntegrationEnabled string `json:"perf_integration_enabled"`
	VcsGroupNameUrl        string `json:"vcs_group_name_url"`
	VcsIntegrationEnabled  string `json:"vcs_integration_enabled"`
	VcsSshPort             string `json:"vcs_ssh_port"`
	VcsToolName            string `json:"vcs_tool_name"`
}

type GerritSettings struct {
	Config            string `json:"config"`
	ReplicationConfig string `json:"replication_config"`
	SshPort           string `json:"ssh_port"`
}

type PerfSettings struct {
	PerfGerritDsId  string `json:"perf_gerrit_ds_id"`
	PerfGitlabDsId  string `json:"perf_gitlab_ds_id"`
	PerfJenkinsDsId string `json:"perf_jenkins_ds_id"`
	PerfNodeId      string `json:"perf_node_id"`
	PerfSonarDsId   string `json:"perf_sonar_ds_id"`
	PerfWebUrl      string `json:"perf_web_url"`
}
