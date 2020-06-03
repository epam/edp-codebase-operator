package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JiraFixVersionSpec defines the desired state of JiraFixVersion
// +k8s:openapi-gen=true
type JiraFixVersionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Commits      []string `json:"commits"`
	Tickets      []string `json:"tickets"`
	CodebaseName string   `json:"codebaseName"`
}

// JiraFixVersionStatus defines the observed state of JiraFixVersion
// +k8s:openapi-gen=true

type JiraFixVersionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
	DetailedMessage string    `json:"detailed_message"`
	FailureCount    int64     `json:"failureCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraFixVersion is the Schema for the jirafixversions API
// +k8s:openapi-gen=true
type JiraFixVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraFixVersionSpec   `json:"spec,omitempty"`
	Status JiraFixVersionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraFixVersionList contains a list of JiraFixVersion
type JiraFixVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JiraFixVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JiraFixVersion{}, &JiraFixVersionList{})
}
