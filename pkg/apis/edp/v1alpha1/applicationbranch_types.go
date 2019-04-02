package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationBranchSpec defines the desired state of ApplicationBranch
// +k8s:openapi-gen=true
type ApplicationBranchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	AppName    string `json:"appName"`
	BranchName string `json:"branchName"`
	FromCommit string `json:"fromCommit"`
}

// ApplicationBranchStatus defines the observed state of ApplicationBranch
// +k8s:openapi-gen=true
type ApplicationBranchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationBranch is the Schema for the applicationbranches API
// +k8s:openapi-gen=true
type ApplicationBranch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationBranchSpec   `json:"spec,omitempty"`
	Status ApplicationBranchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationBranchList contains a list of ApplicationBranch
type ApplicationBranchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationBranch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationBranch{}, &ApplicationBranchList{})
}
