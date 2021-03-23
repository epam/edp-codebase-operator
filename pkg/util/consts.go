package util

const (
	HelmChartDeploymentScriptType = "helm-chart"

	ChartTemplate       = "Chart.tmpl"
	ChartValuesTemplate = "values.tmpl"
	TemplateFolder      = "templates"

	//statuses
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"
	CodebaseLabelKey = "codebase"

	PrivateSShKeyName = "id_rsa"

	//paths
	GerritTemplates   = "/usr/local/bin/templates/gerrit"
	PipelineTemplates = "/usr/local/bin/pipelines"

	ImportStrategy = "import"
	Application    = "application"
	Javascript     = "javascript"

	JenkinsFolderKind            = "JenkinsFolder"
	CDStageDeployKind            = "CDStageDeploy"
	CDStageJenkinsDeploymentKind = "CDStageJenkinsDeployment"
	V2APIVersion                 = "v2.edp.epam.com/v1alpha1"

	ProjectPushedStatus              = "pushed"
	ProjectTemplatesPushedStatus     = "templates_pushed"
	ProjectVersionGoFilePushedStatus = "version_go"
	GitlabCiFilePushedStatus         = "gitlab ci"

	GithubDomain = "https://github.com/epmd-edp"
	LanguageGo   = "Go"

	GitlabCi = "gitlab ci"

	VersioningTypeEDP = "edp"

	HeadBranchesRefSpec   = "refs/heads/*:refs/heads/*"
	RemoteBranchesRefSpec = "refs/remotes/origin/*:refs/heads/*"
	TagsRefSpec           = "refs/tags/*:refs/tags/*"
)
