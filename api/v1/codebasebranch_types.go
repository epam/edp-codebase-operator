package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CodebaseBranchGitStatusBranchCreated = "branch-created"
)

// CodebaseBranchSpec defines the desired state of CodebaseBranch.
type CodebaseBranchSpec struct {
	// Name of Codebase associated with.
	CodebaseName string `json:"codebaseName"`

	// Name of a branch.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	BranchName string `json:"branchName"`

	// FromCommit is a commit hash or branch name.
	// The new branch will be created starting from the selected commit hash or branch name.
	// If a branch name is provided, the new branch will be created from the latest commit of that branch.
	// +optional
	FromCommit string `json:"fromCommit"`

	// Version of the branch. It's required for versioning type "semver".
	// +nullable
	// +optional
	Version *string `json:"version,omitempty"`

	// Flag if branch is used as "release" branch.
	Release bool `json:"release"`

	// Pipelines is a map of pipelines related to the branch.
	// +nullable
	// +optional
	// +kubebuilder:example:={"review": "review-pipeline", "build": "build-pipeline"}
	Pipelines map[string]string `json:"pipelines,omitempty"`
}

// CodebaseBranchStatus defines the observed state of CodebaseBranch.
type CodebaseBranchStatus struct {
	// Information when the last time the action were performed.
	LastTimeUpdated metaV1.Time `json:"lastTimeUpdated"`

	// +nullable
	// +optional
	VersionHistory []string `json:"versionHistory,omitempty"`

	// +nullable
	// +optional
	LastSuccessfulBuild *string `json:"lastSuccessfulBuild,omitempty"`

	// +nullable
	// +optional
	Build *string `json:"build,omitempty"`

	// Specifies a current status of CodebaseBranch.
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

	// Specifies a current state of CodebaseBranch.
	Value string `json:"value"`

	// Amount of times, operator fail to serve with existing CR.
	FailureCount int64 `json:"failureCount"`

	// Specifies a status of action for git.
	Git string `json:"git,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cb
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Result",type="string",JSONPath=".status.result",description="Result of last action"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="The status of codebasebranch"
// +kubebuilder:printcolumn:name="Codebase Name",type="string",JSONPath=".spec.codebaseName",description="Owner of object"
// +kubebuilder:printcolumn:name="Release",type="boolean",JSONPath=".spec.release",description="Is a release branch"
// +kubebuilder:printcolumn:name="Branch",type="string",JSONPath=".spec.branchName",description="Name of branch"

// CodebaseBranch is the Schema for the CodebaseBranches API.
type CodebaseBranch struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseBranchSpec   `json:"spec,omitempty"`
	Status CodebaseBranchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CodebaseBranchList contains a list of CodebaseBranch.
type CodebaseBranchList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []CodebaseBranch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodebaseBranch{}, &CodebaseBranchList{})
}
