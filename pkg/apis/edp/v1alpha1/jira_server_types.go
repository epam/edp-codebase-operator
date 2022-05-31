package v1alpha1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required. Any new fields you add must have json tags for the fields to be serialized.

// JiraServerSpec defines the desired state of JiraServer
type JiraServerSpec struct {
	ApiUrl string `json:"apiUrl"`

	RootUrl string `json:"rootUrl"`

	CredentialName string `json:"credentialName"`
}

// JiraServerStatus defines the observed state of JiraServer
type JiraServerStatus struct {
	// This flag indicates neither JiraServer are initialized and ready to work. Defaults to false.
	Available bool `json:"available"`

	// Information when the last time the action were performed.
	LastTimeUpdated metaV1.Time `json:"last_time_updated"`

	// Specifies a current status of JiraServer.
	Status string `json:"status"`

	// Detailed information regarding action result
	// which were performed
	// +optional
	DetailedMessage string `json:"detailed_message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion

// JiraServer is the Schema for the JiraServers API
type JiraServer struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraServerSpec   `json:"spec,omitempty"`
	Status JiraServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JiraServerList contains a list of JiraServers
type JiraServerList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []JiraServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JiraServer{}, &JiraServerList{})
}
