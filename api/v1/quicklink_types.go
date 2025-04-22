package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuickLinkSpec defines the desired state of QuickLink.
type QuickLinkSpec struct {
	// Type is a quicklink type. It can be default or system. Default value is default.
	// +kubebuilder:validation:Enum=default;system
	// +kubebuilder:default:=default
	Type string `json:"type"`

	// Url specifies a link to the component.
	// +kubebuilder:default=""
	Url string `json:"url"`

	// Icon is a base64 encoded SVG icon of the component.
	Icon string `json:"icon"`

	// Visible specifies whether a component is visible. The default value is true.
	// +kubebuilder:default=true
	Visible bool `json:"visible"`
}

// QuickLinkStatus defines the observed state of QuickLink.
type QuickLinkStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// QuickLink is the Schema for the quicklinks API.
type QuickLink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuickLinkSpec   `json:"spec,omitempty"`
	Status QuickLinkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QuickLinkList contains a list of QuickLink.
type QuickLinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuickLink `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QuickLink{}, &QuickLinkList{})
}
