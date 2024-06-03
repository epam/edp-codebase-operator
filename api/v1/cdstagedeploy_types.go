package v1

import (
	"fmt"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	CDStageDeployStatusFailed    = "failed"
	CDStageDeployStatusRunning   = "running"
	CDStageDeployStatusPending   = "pending"
	CDStageDeployStatusCompleted = "completed"
)

// CDStageDeploySpec defines the desired state of CDStageDeploy.
type CDStageDeploySpec struct {
	// Name of related pipeline
	Pipeline string `json:"pipeline"`

	// Name of related stage
	Stage string `json:"stage"`

	// Specifies a latest available tag
	Tag CodebaseTag `json:"tag"`

	// A list of available tags
	Tags []CodebaseTag `json:"tags"`
}

type CodebaseTag struct {
	Codebase string `json:"codebase"`
	Tag      string `json:"tag"`
}

// CDStageDeployStatus defines the observed state of CDStageDeploy.
type CDStageDeployStatus struct {
	// Specifies a current status of CDStageDeploy.
	// +optional
	// +kubebuilder:validation:Enum=failed;running;pending;completed
	// +kubebuilder:default=pending
	Status string `json:"status"`

	// Descriptive message for current status.
	// +optional
	Message string `json:"message"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=csd,path=cdstagedeployments
// +kubebuilder:storageversion

// CDStageDeploy is the Schema for the CDStageDeployments API.
type CDStageDeploy struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec CDStageDeploySpec `json:"spec,omitempty"`
	// +kubebuilder:default={status:pending}
	// +optional
	Status CDStageDeployStatus `json:"status"`
}

func (in *CDStageDeploy) SetFailedStatus(err error) {
	in.Status.Status = CDStageDeployStatusFailed
	in.Status.Message = err.Error()
}

func (in *CDStageDeploy) GetStageCRName() string {
	return fmt.Sprintf("%s-%s", in.Spec.Pipeline, in.Spec.Stage)
}

// +kubebuilder:object:root=true

// CDStageDeployList contains a list of CDStageDeploy.
type CDStageDeployList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []CDStageDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDStageDeploy{}, &CDStageDeployList{})
}
