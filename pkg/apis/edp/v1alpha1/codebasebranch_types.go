package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CodebaseBranchSpec defines the desired state of CodebaseBranch
// +k8s:openapi-gen=true
type CodebaseBranchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	CodebaseName string  `json:"codebaseName"`
	BranchName   string  `json:"branchName"`
	FromCommit   string  `json:"fromCommit"`
	Version      *string `json:"version,omitempty"`
	Release      bool    `json:"release"`
}

// CodebaseBranchStatus defines the observed state of CodebaseBranch
// +k8s:openapi-gen=true
type CodebaseBranchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	LastTimeUpdated     time.Time  `json:"lastTimeUpdated"`
	VersionHistory      []string   `json:"versionHistory"`
	LastSuccessfulBuild *string    `json:"lastSuccessfulBuild,omitempty"`
	Build               *string    `json:"build,omitempty"`
	Status              string     `json:"status"`
	Username            string     `json:"username"`
	Action              ActionType `json:"action"`
	Result              Result     `json:"result"`
	DetailedMessage     string     `json:"detailedMessage"`
	Value               string     `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CodebaseBranch is the Schema for the codebasebranches API
// +k8s:openapi-gen=true
type CodebaseBranch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseBranchSpec   `json:"spec,omitempty"`
	Status CodebaseBranchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CodebaseBranchList contains a list of CodebaseBranch
type CodebaseBranchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodebaseBranch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodebaseBranch{}, &CodebaseBranchList{})
}
