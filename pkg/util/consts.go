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

	// PrivateSShKeyName is a field in secret created for the git server that stores id_rsa.
	PrivateSShKeyName = "id_rsa"
	// GitServerSecretTokenField is a field in secret created for the git server that stores GitLab/GitHub token.
	GitServerSecretTokenField = "token"
	// GitServerSecretWebhookSecretField is a field in secret created for the git server that stores secret token for webhook.
	GitServerSecretWebhookSecretField = "secretString"

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

	// PauseAnnotation is a key for pause annotation that disables custom resource reconciliation.
	PauseAnnotation = "edp.epam.com/paused"

	// finalizers.
	ForegroundDeletionFinalizerName = "foregroundDeletion"
)
