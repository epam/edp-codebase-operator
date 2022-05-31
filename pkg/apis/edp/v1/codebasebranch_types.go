package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CodebaseBranchSpec defines the desired state of CodebaseBranch
type CodebaseBranchSpec struct {
	// Name of Codebase associated with.
	CodebaseName string `json:"codebaseName"`

	// Name of a branch.
	BranchName string `json:"branchName"`

	// The new branch will be created starting from the selected commit hash.
	FromCommit string `json:"fromCommit"`

	// +nullable
	// +optional
	Version *string `json:"version,omitempty"`

	// Flag if branch is used as "release" branch.
	Release bool `json:"release"`

	// +nullable
	// +optional
	ReleaseJobParams map[string]string `json:"releaseJobParams,omitempty"`
}

// CodebaseBranchStatus defines the observed state of CodebaseBranch
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
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// CodebaseBranch is the Schema for the CodebaseBranches API
type CodebaseBranch struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseBranchSpec   `json:"spec,omitempty"`
	Status CodebaseBranchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CodebaseBranchList contains a list of CodebaseBranch
type CodebaseBranchList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []CodebaseBranch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodebaseBranch{}, &CodebaseBranchList{})
}
