package util

const (
	OpenshiftTemplate = "openshift-template"

	OcTemplatesFolder        = "oc-templates"
	HelmChartTemplatesFolder = "helm-charts"

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
	Autotests      = "autotests"
	Library        = "library"
	OtherLanguage  = "other"
	Javascript     = "javascript"

	CodebaseKind = "Codebase"

	ProjectCreatedStatus         = "created"
	ProjectPushedStatus          = "pushed"
	ProjectTemplatesPushedStatus = "templates_pushed"

	GithubDomain = "https://github.com/epmd-edp"
	LanguageJava = "Java"
)
