package v1alpha1

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	failed  = "failed"
	success = "success"
)

// CDStageDeploySpec defines the desired state of CDStageDeploy
// +k8s:openapi-gen=true
type CDStageDeploySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Pipeline string       `json:"pipeline"`
	Stage    string       `json:"stage"`
	Tag      v1alpha1.Tag `json:"tag"`
}

// CDStageDeployStatus defines the observed state of CDStageDeploy
// +k8s:openapi-gen=true

type CDStageDeployStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Status       string `json:"status"`
	Message      string `json:"message"`
	FailureCount int64  `json:"failureCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CDStageDeploy is the Schema for the cdstagedeploys API
// +k8s:openapi-gen=true
type CDStageDeploy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDStageDeploySpec   `json:"spec,omitempty"`
	Status CDStageDeployStatus `json:"status,omitempty"`
}

func (in *CDStageDeploy) SetFailedStatus(err error) {
	in.Status.Status = failed
	in.Status.Message = err.Error()
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CDStageDeployList contains a list of CDStageDeploy
type CDStageDeployList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDStageDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDStageDeploy{}, &CDStageDeployList{})
}
