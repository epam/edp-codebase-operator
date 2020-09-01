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

	PrivateSShKeyName = "id_rsa"

	//paths
	GerritTemplates   = "/usr/local/bin/templates/gerrit"
	PipelineTemplates = "/usr/local/bin/pipelines"

	ImportStrategy = "import"
	Application    = "application"
	OtherLanguage  = "other"
	Javascript     = "javascript"

	CodebaseKind = "Codebase"

	ProjectPushedStatus              = "pushed"
	ProjectTemplatesPushedStatus     = "templates_pushed"
	ProjectVersionGoFilePushedStatus = "version_go"

	GithubDomain   = "https://github.com/epmd-edp"
	LanguageJava   = "Java"
	LanguageGo     = "Go"
	LanguagePython = "Python"
	LanguageDotnet = "Dotnet"

	GitlabCi = "gitlabci"

	VersioningTypeEDP = "edp"
)
