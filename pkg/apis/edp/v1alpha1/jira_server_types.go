package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JiraServerSpec defines the desired state of JiraServer
// +k8s:openapi-gen=true
type JiraServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ApiUrl         string `json:"apiUrl"`
	RootUrl        string `json:"rootUrl"`
	CredentialName string `json:"credentialName"`
}

// JiraServerStatus defines the observed state of JiraServer
// +k8s:openapi-gen=true

type JiraServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool      `json:"available"`
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
	Username        string    `json:"username"`
	Action          string    `json:"action"`
	Result          string    `json:"result"`
	DetailedMessage string    `json:"detailed_message"`
	Value           string    `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraServer is the Schema for the gitservers API
// +k8s:openapi-gen=true
type JiraServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraServerSpec   `json:"spec,omitempty"`
	Status JiraServerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraServerList contains a list of JiraServer
type JiraServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JiraServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JiraServer{}, &JiraServerList{})
}
