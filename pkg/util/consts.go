package util

const (
	HelmChartDeploymentScriptType = "helm-chart"

	ChartTemplate       = "Chart.tmpl"
	ChartValuesTemplate = "values.tmpl"
	TemplateFolder      = "templates"
	TestFolder          = "tests"
	HelmIgnoreFile      = ".helmignore"
	TestFile            = "test-connection.yaml"

	// statuses.
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"
	CodebaseLabelKey = "codebase"

	PrivateSShKeyName = "id_rsa"

	ImportStrategy     = "import"
	Application        = "application"
	LanguageJavascript = "javascript"
	LanguagePython     = "python"
	LanguageGo         = "go"

	JenkinsFolderKind            = "JenkinsFolder"
	CDStageDeployKind            = "CDStageDeploy"
	CDStageJenkinsDeploymentKind = "CDStageJenkinsDeployment"
	V2APIVersion                 = "v2.edp.epam.com/v1"

	ProjectPushedStatus              = "pushed"
	ProjectTemplatesPushedStatus     = "templates_pushed"
	ProjectVersionGoFilePushedStatus = "version_go"
	GitlabCiFilePushedStatus         = "gitlab ci"

	GithubDomain = "https://github.com/epmd-edp"

	GitlabCi = "gitlab ci"

	Tekton = "tekton"

	VersioningTypeEDP = "edp"

	// finalizers.
	ForegroundDeletionFinalizerName = "foregroundDeletion"
)
