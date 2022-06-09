package v1

import (
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	failed = "failed"
)

// CDStageDeploySpec defines the desired state of CDStageDeploy
type CDStageDeploySpec struct {
	// Name of related pipeline
	Pipeline string `json:"pipeline"`

	// Name of related stage
	Stage string `json:"stage"`

	// Specifies a latest available tag
	Tag jenkinsApi.Tag `json:"tag"`

	// A list of available tags
	Tags []jenkinsApi.Tag `json:"tags"`
}

// CDStageDeployStatus defines the observed state of CDStageDeploy
type CDStageDeployStatus struct {
	// Specifies a current status of CDStageDeploy.
	Status string `json:"status"`

	// Descriptive message for current status
	Message string `json:"message"`

	// Amount of times, operator fail to serve with existing CR
	FailureCount int64 `json:"failureCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=csd,path=cdstagedeployments
// +kubebuilder:storageversion

// CDStageDeploy is the Schema for the CDStageDeployments API
type CDStageDeploy struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDStageDeploySpec   `json:"spec,omitempty"`
	Status CDStageDeployStatus `json:"status,omitempty"`
}

func (in *CDStageDeploy) SetFailedStatus(err error) {
	in.Status.Status = failed
	in.Status.Message = err.Error()
}

// +kubebuilder:object:root=true

// CDStageDeployList contains a list of CDStageDeploy
type CDStageDeployList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []CDStageDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDStageDeploy{}, &CDStageDeployList{})
}
