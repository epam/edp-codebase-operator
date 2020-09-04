package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GitTagSpec defines the desired state of GitTag
// +k8s:openapi-gen=true
type GitTagSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Codebase string `json:"codebase"`
	Branch   string `json:"branch"`
	Tag      string `json:"tag"`
}

// GitTagStatus defines the observed state of GitTag
// +k8s:openapi-gen=true
type GitTagStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitTag is the Schema for the gittags API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type GitTag struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitTagSpec   `json:"spec,omitempty"`
	Status GitTagStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitTagList contains a list of GitTag
type GitTagList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitTag `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitTag{}, &GitTagList{})
}
