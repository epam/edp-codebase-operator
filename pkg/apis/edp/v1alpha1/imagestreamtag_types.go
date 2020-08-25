package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageStreamTagSpec defines the desired state of ImageStreamTag
// +k8s:openapi-gen=true
type ImageStreamTagSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	CodebaseImageStreamName string `json:"codebaseImageStreamName"`
	Tag                     string `json:"tag"`
}

// ImageStreamTagStatus defines the observed state of ImageStreamTag
// +k8s:openapi-gen=true
type ImageStreamTagStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageStreamTag is the Schema for the imagestreamtags API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type ImageStreamTag struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageStreamTagSpec   `json:"spec,omitempty"`
	Status ImageStreamTagStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageStreamTagList contains a list of ImageStreamTag
type ImageStreamTagList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageStreamTag `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageStreamTag{}, &ImageStreamTagList{})
}
