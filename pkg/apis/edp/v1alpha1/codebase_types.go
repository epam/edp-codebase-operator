package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CodebaseSpec defines the desired state of Codebase
// +k8s:openapi-gen=true
const (
	Create Strategy = "create"
	Clone  Strategy = "clone"
)

type Strategy string

type Repository struct {
	Url string `json:"url"`
}

type Route struct {
	Site string `json:"site"`
	Path string `json:"path"`
}

type Database struct {
	Kind     string `json:"kind"`
	Version  string `json:"version"`
	Capacity string `json:"capacity"`
	Storage  string `json:"storage"`
}

type CodebaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Lang                string      `json:"lang"`
	Description         *string     `json:"description"`
	Framework           string      `json:"framework"`
	BuildTool           string      `json:"buildTool"`
	Strategy            Strategy    `json:"strategy"`
	Repository          *Repository `json:"repository"`
	Route               *Route      `json:"route"`
	Database            *Database   `json:"database"`
	TestReportFramework *string     `json:"testReportFramework"`
	Type                string      `json:"type"`
}

// CodebaseStatus defines the observed state of Codebase
// +k8s:openapi-gen=true
type CodebaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool      `json:"available"`
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Codebase is the Schema for the codebases API
// +k8s:openapi-gen=true
type Codebase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseSpec   `json:"spec,omitempty"`
	Status CodebaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CodebaseList contains a list of Codebase
type CodebaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Codebase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Codebase{}, &CodebaseList{})
}
