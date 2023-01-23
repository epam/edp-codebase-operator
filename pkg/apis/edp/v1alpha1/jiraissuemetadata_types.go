package v1alpha1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JiraIssueMetadataSpec defines the desired state of JiraIssueMetadata.
type JiraIssueMetadataSpec struct {
	// Name of Codebase associated with.
	CodebaseName string `json:"codebaseName"`

	// JSON payload
	// +optional
	Payload string `json:"payload,omitempty"`

	// +nullable
	// +optional
	Commits []string `json:"commits,omitempty"`

	// +nullable
	// +optional
	Tickets []string `json:"tickets,omitempty"`
}

// JiraIssueMetadataStatus defines the observed state of JiraIssueMetadata.
type JiraIssueMetadataStatus struct {
	// Information when the last time the action were performed.
	LastTimeUpdated metaV1.Time `json:"last_time_updated"`

	// Specifies a current status of JiraIssueMetadata.
	Status string `json:"status"`

	// Detailed information regarding action result
	// which were performed
	// +optional
	DetailedMessage string `json:"detailed_message,omitempty"`

	// Amount of times, operator fail to serve with existing CR.
	FailureCount int64 `json:"failureCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=jim,path=jiraissuemetadatas
// +kubebuilder:deprecatedversion

// JiraIssueMetadata is the Schema for the JiraIssueMetadatas API.
type JiraIssueMetadata struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraIssueMetadataSpec   `json:"spec,omitempty"`
	Status JiraIssueMetadataStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JiraIssueMetadataList contains a list of JiraIssueMetadata.
type JiraIssueMetadataList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []JiraIssueMetadata `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JiraIssueMetadata{}, &JiraIssueMetadataList{})
}
