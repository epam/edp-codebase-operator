package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required. Any new fields you add must have json tags for the fields to be serialized.

// GitTagSpec defines the desired state of GitTag.
type GitTagSpec struct {
	// Name of Codebase associated with.
	Codebase string `json:"codebase"`

	// Name of a branch.
	Branch string `json:"branch"`

	// Name of a git tag.
	Tag string `json:"tag"`
}

// GitTagStatus defines the observed state of GitTag.
type GitTagStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gt
// +kubebuilder:storageversion

// GitTag is the Schema for the GitTags API.
type GitTag struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitTagSpec   `json:"spec,omitempty"`
	Status GitTagStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitTagList contains a list of GitTags.
type GitTagList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []GitTag `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitTag{}, &GitTagList{})
}
