package v1

const (
	// GitLabCITemplateAnnotation is an annotation on a Codebase CR that specifies the ConfigMap
	// name to use as the GitLab CI template. When absent, the operator falls back to "gitlab-ci-default".
	GitLabCITemplateAnnotation = "app.edp.epam.com/gitlab-ci-template"
)
