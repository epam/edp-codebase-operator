package v1alpha1

import (
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
	Default VersioningType = "default"
)

type Versioning struct {
	Type VersioningType `json:"type"`

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
	// +nullable
	// +optional
	Framework *string `json:"framework,omitempty"`

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

	// A link to external git server, used for "import" strategy.
	// +nullable
	// +optional
	GitUrlPath *string `json:"gitUrlPath,omitempty"`

	// +optional
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

type ActionType string

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
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cdbs
// +kubebuilder:deprecatedversion

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
