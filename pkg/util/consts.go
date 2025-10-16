package util

const (
	HelmChartDeploymentScriptType  = "helm-chart"
	RpmPackageDeploymentScriptType = "rpm-package"

	ChartTemplate       = "Chart.tmpl"
	ReadmeTemplate      = "README.tmpl"
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
	CloneStrategy      = "clone"
	Application        = "application"
	LanguageJavascript = "javascript"
	LanguagePython     = "python"
	LanguageGo         = "go"

	CDStageDeployKind = "CDStageDeploy"
	V2APIVersion      = "v2.edp.epam.com/v1"

	ProjectPushedStatus          = "pushed"
	ProjectGitLabCIPushedStatus  = "gitlab_ci_pushed"
	ProjectTemplatesPushedStatus = "templates_pushed"

	GithubDomain = "https://github.com/epmd-edp"

	CITekton = "tekton"
	CIGitLab = "gitlab"

	// finalizers.
	ForegroundDeletionFinalizerName = "foregroundDeletion"
)
