package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageStreamTagSpec defines the desired state of ImageStreamTag.
type ImageStreamTagSpec struct {
	CodebaseImageStreamName string `json:"codebaseImageStreamName"`

	Tag string `json:"tag"`
}

// ImageStreamTagStatus defines the observed state of ImageStreamTag.
type ImageStreamTagStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ist
// +kubebuilder:storageversion

// ImageStreamTag is the Schema for the ImageStreamTags API.
type ImageStreamTag struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageStreamTagSpec   `json:"spec,omitempty"`
	Status ImageStreamTagStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageStreamTagList contains a list of ImageStreamTag.
type ImageStreamTagList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []ImageStreamTag `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageStreamTag{}, &ImageStreamTagList{})
}
