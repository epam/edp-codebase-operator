package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TemplateSpec defines the desired state of Template.
type TemplateSpec struct {

	// The name of the template.
	DisplayName string `json:"displayName"`

	// The build tool used to build the component from the template.
	BuildTool string `json:"buildTool"`

	// The type of the template, e.g application, library, autotest, infrastructure, etc.
	Type string `json:"type"`

	// The framework used to build the component from the template.
	Framework string `json:"framework"`

	// The programming language used to build the component from the template.
	Language string `json:"language"`

	// The description of the template.
	Description string `json:"description"`

	// Category is the category of the template.
	// +optional
	Category string `json:"category,omitempty"`

	// The icon for this template.
	// +optional
	// +kubebuilder:validation:MaxItems=1
	Icon []Icon `json:"icon,omitempty"`

	// A list of keywords describing the template.
	// +optional
	Keywords []string `json:"keywords,omitempty"`

	// A list of organizational entities maintaining the Template.
	// +optional
	Maintainers []Maintainer `json:"maintainers,omitempty"`

	// A repository containing the source code for the template.
	Source string `json:"source"`

	// Version is the version of the template.
	Version string `json:"version"`

	// MinEDPVersion is the minimum EDP version that this template is compatible with.
	// +optional
	MinEDPVersion string `json:"minEDPVersion,omitempty"`

	// The level of maturity the template has achieved at this version. Options include planning, pre-alpha, alpha, beta, stable, mature, inactive, and deprecated.
	// +kubebuilder:validation:Enum=planning;pre-alpha;alpha;beta;stable;mature;inactive;deprecated
	Maturity string `json:"maturity,omitempty"`
}

type Icon struct {
	// A base64 encoded PNG, JPEG or SVG image.
	Base64Data string `json:"base64data"`
	// The media type of the image. E.g image/svg+xml, image/png, image/jpeg.
	MediaType string `json:"mediatype"`
}

type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TemplateStatus defines the observed state of Template.
type TemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="Template version"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="Codebase type"
// +kubebuilder:printcolumn:name="Framework",type="string",JSONPath=".spec.framework",description="Framework"
// +kubebuilder:printcolumn:name="Language",type="string",JSONPath=".spec.language",description="Language"
// +kubebuilder:printcolumn:name="BuildTool",type="string",JSONPath=".spec.buildTool",description="Build tool"

// Template is the Schema for the templates API.
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateSpec   `json:"spec,omitempty"`
	Status TemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TemplateList contains a list of Template.
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}
