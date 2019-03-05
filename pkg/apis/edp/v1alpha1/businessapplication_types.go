package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BusinessApplicationSpec defines the desired state of BusinessApplication
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

type BusinessApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Lang       string      `json:"lang"`
	Framework  string      `json:"framework"`
	BuildTool  string      `json:"buildTool"`
	Strategy   Strategy    `json:"strategy"`
	Repository *Repository `json:"repository"`
	Route      *Route      `json:"route"`
	Database   *Database   `json:"database"`
}

// BusinessApplicationStatus defines the observed state of BusinessApplication
// +k8s:openapi-gen=true
type BusinessApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool      `json:"available"`
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BusinessApplication is the Schema for the businessapplications API
// +k8s:openapi-gen=true
type BusinessApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessApplicationSpec   `json:"spec,omitempty"`
	Status BusinessApplicationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BusinessApplicationList contains a list of BusinessApplication
type BusinessApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BusinessApplication{}, &BusinessApplicationList{})
}
