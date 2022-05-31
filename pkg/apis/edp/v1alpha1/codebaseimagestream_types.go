package v1alpha1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CodebaseImageStreamSpec defines the desired state of CodebaseImageStream
type CodebaseImageStreamSpec struct {
	// Name of Codebase associated with.
	Codebase string `json:"codebase"`

	// Docker container name without tag, e.g. registry-name/path/name.
	ImageName string `json:"imageName"`

	// A list of docker image tags available for ImageName and their creation date.
	// +nullable
	// +optional
	Tags []Tag `json:"tags,omitempty"`
}

type Tag struct {
	Name    string `json:"name"`
	Created string `json:"created"`
}

// CodebaseImageStreamStatus defines the observed state of CodebaseImageStream
type CodebaseImageStreamStatus struct {
	// Detailed information regarding action result
	// which were performed
	// +optional
	DetailedMessage string `json:"detailed_message"`

	// Amount of times, operator fail to serve with existing CR.
	FailureCount int64 `json:"failureCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion

// CodebaseImageStream is the Schema for the CodebaseImageStreams API
type CodebaseImageStream struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseImageStreamSpec   `json:"spec,omitempty"`
	Status CodebaseImageStreamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CodebaseImageStreamList contains a list of CodebaseImageStreams
type CodebaseImageStreamList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []CodebaseImageStream `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodebaseImageStream{}, &CodebaseImageStreamList{})
}
