package v1

import (
	"fmt"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CDStageDeployStatusFailed    = "failed"
	CDStageDeployStatusRunning   = "running"
	CDStageDeployStatusPending   = "pending"
	CDStageDeployStatusInQueue   = "in-queue"
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
	// +kubebuilder:validation:Enum=failed;running;pending;completed;in-queue
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
// +kubebuilder:printcolumn:name="Pipeline",type="string",JSONPath=".spec.pipeline",description="Pipeline name"
// +kubebuilder:printcolumn:name="Stage",type="string",JSONPath=".spec.stage",description="Stage name"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="Pipeline status"

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
