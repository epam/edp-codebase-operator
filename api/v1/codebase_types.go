package v1

import (
	"strings"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Strategy describes integration strategy for a codebase.
// +kubebuilder:validation:Enum=create;clone;import
type Strategy string

const (
	// Create a new codebase.
	Create Strategy = "create"

	// Clone an existing codebase.
	Clone Strategy = "clone"

	// Import existing codebase.
	Import Strategy = "import"
)

type VersioningType string

const (
	VersioningTypDefault VersioningType = "default"
	VersioningTypeEDP    VersioningType = "edp"
)

type Versioning struct {
	Type VersioningType `json:"type"`

	// StartFrom is required when versioning type is not default.
	// +nullable
	// +optional
	StartFrom *string `json:"startFrom,omitempty"`
}

type Repository struct {
	Url string `json:"url"`
}

// CodebaseSpec defines the desired state of Codebase.
type CodebaseSpec struct {
	// Programming language used in codebase.
	Lang string `json:"lang"`

	// A short description of codebase.
	// +nullable
	// +optional
	Description *string `json:"description,omitempty"`

	// A framework used in codebase.
	Framework string `json:"framework"`

	// A build tool which is used on codebase.
	BuildTool string `json:"buildTool"`

	// integration strategy for a codebase, e.g. clone, import, etc.
	Strategy Strategy `json:"strategy"`

	// +nullable
	// +optional
	Repository *Repository `json:"repository,omitempty"`

	// +nullable
	// +optional
	TestReportFramework *string `json:"testReportFramework,omitempty"`

	// Type of codebase. E.g. application, autotest or library.
	Type string `json:"type"`

	// A name of git server which will be used as VCS.
	// Example: "gerrit".
	GitServer string `json:"gitServer"`

	// A relative path for git repository. Should start from /. Example: /company/api-app.
	GitUrlPath string `json:"gitUrlPath"`

	// +optional
	// +kubebuilder:default:=helm-chart
	DeploymentScript string `json:"deploymentScript,omitempty"`

	Versioning Versioning `json:"versioning"`

	// +nullable
	// +optional
	JiraServer *string `json:"jiraServer,omitempty"`

	// +nullable
	// +optional
	CommitMessagePattern *string `json:"commitMessagePattern,omitempty"`

	// +nullable
	// +optional
	TicketNamePattern *string `json:"ticketNamePattern"`

	// A name of tool which should be used as CI.
	// +optional
	// +kubebuilder:default:=tekton
	CiTool string `json:"ciTool"`

	// Name of default branch.
	DefaultBranch string `json:"defaultBranch"`

	// +nullable
	// +optional
	JiraIssueMetadataPayload *string `json:"jiraIssueMetadataPayload"`

	// A flag indicating how project should be provisioned. Default: false
	EmptyProject bool `json:"emptyProject"`

	// While we clone new codebase we can select specific branch to clone.
	// Selected branch will become a default branch for a new codebase (e.g. master, main).
	// +optional
	BranchToCopyInDefaultBranch string `json:"branchToCopyInDefaultBranch,omitempty"`

	// Controller must skip step "put deploy templates" in action chain.
	// +optional
	DisablePutDeployTemplates bool `json:"disablePutDeployTemplates,omitempty"`
}

// GetProjectID returns project id from GitUrlPath codebase spec. It removes the leading slash.
func (in *CodebaseSpec) GetProjectID() string {
	return strings.TrimPrefix(in.GitUrlPath, "/")
}

type ActionType string

const (
	GerritRepositoryProvisioning     ActionType = "gerrit_repository_provisioning"
	CIConfiguration                  ActionType = "ci_configuration"
	SetupDeploymentTemplates         ActionType = "setup_deployment_templates"
	AcceptCodebaseBranchRegistration ActionType = "accept_codebase_branch_registration"
	CleanData                        ActionType = "clean_data"
	PutWebHook                       ActionType = "put_web_hook"
	PutGitWebRepoUrl                 ActionType = "put_git_web_repo_url"
	PutGitBranch                     ActionType = "put_git_branch"
	PutCodebaseImageStream           ActionType = "put_codebase_image_stream"
	CheckCommitHashExists            ActionType = "check_commit_hash_exists"
)

// Result describes how action were performed.
// Once action ended, we record a result of this action.
// +kubebuilder:validation:Enum=success;error
type Result string

const (
	// Success result of operation.
	Success Result = "success"

	// Error result point to unsuccessful operation.
	Error Result = "error"
)

// CodebaseStatus defines the observed state of Codebase.
type CodebaseStatus struct {
	// This flag indicates neither Codebase are initialized and ready to work. Defaults to false.
	Available bool `json:"available"`

	// Information when the last time the action were performed.
	LastTimeUpdated metaV1.Time `json:"lastTimeUpdated"`

	// Specifies a current status of Codebase.
	Status string `json:"status"`

	// Name of user who made a last change.
	Username string `json:"username"`

	// The last Action was performed.
	Action ActionType `json:"action"`

	// A result of an action which were performed.
	// - "success": action where performed successfully;
	// - "error": error has occurred;
	Result Result `json:"result"`

	// Detailed information regarding action result
	// which were performed
	// +optional
	DetailedMessage string `json:"detailedMessage,omitempty"`

	// Specifies a current state of Codebase.
	Value string `json:"value"`

	// Amount of times, operator fail to serve with existing CR.
	FailureCount int64 `json:"failureCount"`

	// Specifies a status of action for git.
	Git string `json:"git"`

	// Stores ID of webhook which was created for a codebase.
	// +optional
	WebHookID int `json:"webHookID,omitempty"`

	// Stores GitWebUrl of codebase.
	// +optional
	GitWebUrl string `json:"gitWebUrl,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cdbs
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="Codebase type"
// +kubebuilder:printcolumn:name="Available",type="boolean",JSONPath=".status.available",description="Is resource available"
// +kubebuilder:printcolumn:name="Result",type="string",JSONPath=".status.result",description="Result of last action"
// +kubebuilder:printcolumn:name="Default Branch",type="string",JSONPath=".spec.defaultBranch",description="Default Branch"
// +kubebuilder:printcolumn:name="GitWebUrl",type="string",JSONPath=".status.gitWebUrl",description="Repository Web URL"

// Codebase is the Schema for the Codebases API.
type Codebase struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseSpec   `json:"spec,omitempty"`
	Status CodebaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CodebaseList contains a list of Codebases.
type CodebaseList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []Codebase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Codebase{}, &CodebaseList{})
}
