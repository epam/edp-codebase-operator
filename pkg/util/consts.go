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
	// GitServerSecretWebhookSecretField is a field in secret created for the git server that stores
	// secret token for webhook.
	GitServerSecretWebhookSecretField = "secretString"
	// GitServerSecretUserNameField is a field in secret created for the git server that stores username.
	GitServerSecretUserNameField = "username"

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
	// ProjectPushInProgressStatus marks that the operator verified the remote
	// default branch was absent and is about to push the initial history. On
	// retry, this status plus a now-present remote default branch proves the
	// push landed and provisioning must adopt it: pushing regenerated history
	// would silently replace the remote branch.
	ProjectPushInProgressStatus = "push_in_progress"

	GithubDomain = "https://github.com/epmd-edp"

	CITekton = "tekton"
	CIGitLab = "gitlab"

	// finalizers.
	ForegroundDeletionFinalizerName = "foregroundDeletion"
)
