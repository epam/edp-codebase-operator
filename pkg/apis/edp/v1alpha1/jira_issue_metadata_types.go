package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JiraIssueMetadataSpec defines the desired state of JiraIssueMetadata
// +k8s:openapi-gen=true
type JiraIssueMetadataSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Commits      []string `json:"commits"`
	Tickets      []string `json:"tickets"`
	CodebaseName string   `json:"codebaseName"`
	Payload      string   `json:"payload"`
}

// JiraIssueMetadataStatus defines the observed state of JiraIssueMetadata
// +k8s:openapi-gen=true

type JiraIssueMetadataStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
	DetailedMessage string    `json:"detailed_message"`
	FailureCount    int64     `json:"failureCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraIssueMetadata is the Schema for the jiraissuemetadatas API
// +k8s:openapi-gen=true
type JiraIssueMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraIssueMetadataSpec   `json:"spec,omitempty"`
	Status JiraIssueMetadataStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraIssueMetadataList contains a list of JiraIssueMetadata
type JiraIssueMetadataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JiraIssueMetadata `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JiraIssueMetadata{}, &JiraIssueMetadataList{})
}
